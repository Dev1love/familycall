package handlers

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"time"

	"familycall/server/internal/models"
	"familycall/server/internal/websocket"

	"github.com/gin-gonic/gin"
)

// ListMessages returns messages for a chat with cursor-based pagination.
// GET /api/chats/:id/messages?cursor=&limit=50
func (h *Handlers) ListMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	chatID := c.Param("id")

	// Check membership
	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", chatID, uid).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	query := h.db.Where("chat_id = ?", chatID)

	cursor := c.Query("cursor")
	if cursor != "" {
		var cursorMsg models.Message
		if err := h.db.Select("created_at").Where("id = ?", cursor).First(&cursorMsg).Error; err == nil {
			query = query.Where("created_at < ?", cursorMsg.CreatedAt)
		}
	}

	var messages []models.Message
	if err := query.Preload("Sender").Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}

	// Update last_read_at
	now := time.Now()
	h.db.Model(&models.ChatMember{}).Where("chat_id = ? AND user_id = ?", chatID, uid).Update("last_read_at", now)

	c.JSON(http.StatusOK, messages)
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// SendMessage creates a new message in a chat and broadcasts via WebSocket.
// POST /api/chats/:id/messages
func (h *Handlers) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	chatID := c.Param("id")

	// Check membership
	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", chatID, uid).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := models.Message{
		ChatID:   chatID,
		SenderID: uid,
		Content:  html.EscapeString(req.Content),
	}

	if err := h.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		return
	}

	// Update chat's updated_at
	h.db.Model(&models.Chat{}).Where("id = ?", chatID).Update("updated_at", time.Now())

	// Update sender's last_read_at
	h.db.Model(&models.ChatMember{}).Where("chat_id = ? AND user_id = ?", chatID, uid).Update("last_read_at", time.Now())

	// Reload with sender
	h.db.Preload("Sender").First(&msg, "id = ?", msg.ID)

	// Broadcast to other chat members via WebSocket
	var members []models.ChatMember
	h.db.Where("chat_id = ? AND user_id != ?", chatID, uid).Find(&members)

	for _, m := range members {
		h.hub.SendToUser(m.UserID, websocket.Message{
			Type: "chat:message",
			From: uid,
			To:   m.UserID,
			Data: map[string]interface{}{
				"message": msg,
				"chat_id": chatID,
			},
		})
	}

	// Send push notifications to offline members
	go func() {
		// Load chat info for notification title
		var chat models.Chat
		if err := h.db.First(&chat, "id = ?", chatID).Error; err != nil {
			log.Printf("[PUSH] Failed to load chat %s for push notification: %v", chatID, err)
			return
		}

		senderName := msg.Sender.Username

		for _, m := range members {
			if h.hub.IsUserOnline(m.UserID) {
				continue
			}

			var title, body string
			if chat.Type == "direct" {
				title = senderName
				body = msg.Content
			} else {
				// Group chat
				if chat.Name != nil && *chat.Name != "" {
					title = *chat.Name
				} else {
					title = "Group chat"
				}
				body = fmt.Sprintf("%s: %s", senderName, msg.Content)
			}

			// Truncate body to 200 chars
			if len(body) > 200 {
				body = body[:197] + "..."
			}

			data := map[string]interface{}{
				"type": "chat_message",
				"url":  fmt.Sprintf("/chat/%s", chatID),
			}

			if err := h.SendPushNotification(m.UserID, title, body, data); err != nil {
				log.Printf("[PUSH] Failed to send chat push to user %s: %v", m.UserID, err)
			}
		}
	}()

	c.JSON(http.StatusCreated, msg)
}

type EditMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// EditMessage updates the content of a message owned by the authenticated user.
// PUT /api/messages/:id
func (h *Handlers) EditMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	messageID := c.Param("id")

	var msg models.Message
	if err := h.db.First(&msg, "id = ?", messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	if msg.SenderID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only edit own messages"})
		return
	}

	// Check membership
	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", msg.ChatID, uid).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	var req EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	h.db.Model(&msg).Updates(map[string]interface{}{
		"content":   html.EscapeString(req.Content),
		"edited_at": now,
	})

	h.db.Preload("Sender").First(&msg, "id = ?", msg.ID)
	c.JSON(http.StatusOK, msg)
}

// DeleteMessage deletes a message owned by the authenticated user.
// DELETE /api/messages/:id
func (h *Handlers) DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	messageID := c.Param("id")

	var msg models.Message
	if err := h.db.First(&msg, "id = ?", messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	if msg.SenderID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only delete own messages"})
		return
	}

	// Check membership
	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", msg.ChatID, uid).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	if err := h.db.Delete(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
