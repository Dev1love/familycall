package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"familycall/server/internal/models"
	"familycall/server/internal/websocket"

	"github.com/gin-gonic/gin"
)

// StartGroupCall creates a new group call in a chat and invites all members.
// POST /api/chats/:id/calls
func (h *Handlers) StartGroupCall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	chatID := c.Param("id")

	// Check membership
	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", chatID, uid).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	// Check no active call exists for this chat
	var activeCall models.GroupCall
	if err := h.db.Where("chat_id = ? AND ended_at IS NULL", chatID).First(&activeCall).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "an active call already exists", "call_id": activeCall.ID})
		return
	}

	// Create GroupCall record
	now := time.Now()
	call := models.GroupCall{
		ChatID:    chatID,
		StartedBy: uid,
		StartedAt: now,
	}
	if err := h.db.Create(&call).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create call"})
		return
	}

	// Add starter as participant
	participant := models.GroupCallParticipant{
		CallID:   call.ID,
		UserID:   uid,
		JoinedAt: now,
	}
	if err := h.db.Create(&participant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add participant"})
		return
	}

	// Get starter user for notification
	var starter models.User
	h.db.First(&starter, "id = ?", uid)

	// Get all other chat members
	var members []models.ChatMember
	h.db.Where("chat_id = ? AND user_id != ?", chatID, uid).Find(&members)

	// Send call:group_invite via WebSocket to all other chat members
	for _, m := range members {
		h.hub.SendToUser(m.UserID, websocket.Message{
			Type: websocket.TypeGroupCallInvite,
			From: uid,
			To:   m.UserID,
			Data: map[string]interface{}{
				"call_id": call.ID,
				"chat_id": chatID,
				"starter": starter.Username,
			},
		})
	}

	// Send push notification to offline members
	go func() {
		var chat models.Chat
		if err := h.db.First(&chat, "id = ?", chatID).Error; err != nil {
			log.Printf("[PUSH] Failed to load chat %s for group call push: %v", chatID, err)
			return
		}

		title := "Group Call"
		if chat.Name != nil && *chat.Name != "" {
			title = *chat.Name
		}
		body := fmt.Sprintf("%s started a group call", starter.Username)

		for _, m := range members {
			if h.hub.IsUserOnline(m.UserID) {
				continue
			}

			data := map[string]interface{}{
				"type":    "group_call_invite",
				"call_id": call.ID,
				"chat_id": chatID,
			}

			if err := h.SendPushNotification(m.UserID, title, body, data); err != nil {
				log.Printf("[PUSH] Failed to send group call push to user %s: %v", m.UserID, err)
			}
		}
	}()

	// Reload with associations
	h.db.Preload("Starter").Preload("Participants.User").First(&call, "id = ?", call.ID)

	c.JSON(http.StatusCreated, call)
}

// JoinGroupCall adds a user to an active group call.
// POST /api/calls/:id/join
func (h *Handlers) JoinGroupCall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	callID := c.Param("id")

	// Find the call, verify not ended
	var call models.GroupCall
	if err := h.db.First(&call, "id = ?", callID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "call not found"})
		return
	}
	if call.EndedAt != nil {
		c.JSON(http.StatusGone, gin.H{"error": "call has ended"})
		return
	}

	// Check user is member of the chat
	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", call.ChatID, uid).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	// Add or update participant (set left_at to nil if rejoining)
	now := time.Now()
	var existing models.GroupCallParticipant
	if err := h.db.Where("call_id = ? AND user_id = ?", callID, uid).First(&existing).Error; err == nil {
		// Rejoining — clear left_at
		h.db.Model(&existing).Updates(map[string]interface{}{
			"left_at":   nil,
			"joined_at": now,
		})
	} else {
		// New participant
		p := models.GroupCallParticipant{
			CallID:   callID,
			UserID:   uid,
			JoinedAt: now,
		}
		if err := h.db.Create(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join call"})
			return
		}
	}

	// Get list of current active participants (left_at IS NULL, not the joiner)
	var participants []models.GroupCallParticipant
	h.db.Preload("User").Where("call_id = ? AND left_at IS NULL AND user_id != ?", callID, uid).Find(&participants)

	// Send call:group_join to each current participant
	for _, p := range participants {
		h.hub.SendToUser(p.UserID, websocket.Message{
			Type: websocket.TypeGroupCallJoin,
			From: uid,
			To:   p.UserID,
			Data: map[string]interface{}{
				"call_id": callID,
				"user_id": uid,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"call_id":      callID,
		"participants": participants,
	})
}

// LeaveGroupCall removes a user from a group call and ends it if no one remains.
// POST /api/calls/:id/leave
func (h *Handlers) LeaveGroupCall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)
	callID := c.Param("id")

	// Set left_at on the participant record
	now := time.Now()
	result := h.db.Model(&models.GroupCallParticipant{}).
		Where("call_id = ? AND user_id = ? AND left_at IS NULL", callID, uid).
		Update("left_at", now)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not an active participant"})
		return
	}

	// Count remaining participants
	var remaining int64
	h.db.Model(&models.GroupCallParticipant{}).
		Where("call_id = ? AND left_at IS NULL", callID).
		Count(&remaining)

	// If 0 remaining, set ended_at on the call
	if remaining == 0 {
		h.db.Model(&models.GroupCall{}).Where("id = ?", callID).Update("ended_at", now)
	}

	// Send call:group_leave to all remaining participants
	var activeParticipants []models.GroupCallParticipant
	h.db.Where("call_id = ? AND left_at IS NULL", callID).Find(&activeParticipants)

	for _, p := range activeParticipants {
		h.hub.SendToUser(p.UserID, websocket.Message{
			Type: websocket.TypeGroupCallLeave,
			From: uid,
			To:   p.UserID,
			Data: map[string]interface{}{
				"call_id": callID,
				"user_id": uid,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"left":      true,
		"remaining": remaining,
	})
}
