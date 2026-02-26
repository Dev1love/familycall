package handlers

import (
	"net/http"
	"time"

	"familycall/server/internal/models"

	"github.com/gin-gonic/gin"
)

type ChatListItem struct {
	models.Chat
	LastMessage *models.Message `json:"last_message,omitempty"`
	UnreadCount int64           `json:"unread_count"`
}

// ListChats returns all chats for the authenticated user with last message and unread count.
func (h *Handlers) ListChats(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var memberships []models.ChatMember
	if err := h.db.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chats"})
		return
	}

	chatIDs := make([]string, len(memberships))
	lastReadMap := make(map[string]time.Time)
	for i, m := range memberships {
		chatIDs[i] = m.ChatID
		lastReadMap[m.ChatID] = m.LastReadAt
	}

	if len(chatIDs) == 0 {
		c.JSON(http.StatusOK, []ChatListItem{})
		return
	}

	var chats []models.Chat
	if err := h.db.Preload("Members.User").Where("id IN ?", chatIDs).Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chats"})
		return
	}

	result := make([]ChatListItem, 0, len(chats))
	for _, chat := range chats {
		item := ChatListItem{Chat: chat}

		var lastMsg models.Message
		if err := h.db.Where("chat_id = ?", chat.ID).Preload("Sender").Order("created_at DESC").First(&lastMsg).Error; err == nil {
			item.LastMessage = &lastMsg
		}

		lastRead := lastReadMap[chat.ID]
		h.db.Model(&models.Message{}).Where("chat_id = ? AND created_at > ? AND sender_id != ?", chat.ID, lastRead, userID).Count(&item.UnreadCount)

		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

type CreateChatRequest struct {
	Name      string   `json:"name" binding:"required"`
	MemberIDs []string `json:"member_ids" binding:"required,min=1"`
}

// CreateChat creates a new group chat with the given name and members.
func (h *Handlers) CreateChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	var req CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chat := models.Chat{
		Type:      "group",
		Name:      &req.Name,
		CreatedBy: uid,
	}

	if err := h.db.Create(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat"})
		return
	}

	now := time.Now()
	members := []models.ChatMember{
		{ChatID: chat.ID, UserID: uid, Role: "admin", JoinedAt: now, LastReadAt: now},
	}

	for _, memberID := range req.MemberIDs {
		if memberID != uid {
			members = append(members, models.ChatMember{
				ChatID: chat.ID, UserID: memberID, Role: "member", JoinedAt: now, LastReadAt: now,
			})
		}
	}

	if err := h.db.Create(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add members"})
		return
	}

	h.db.Preload("Members.User").First(&chat, "id = ?", chat.ID)
	c.JSON(http.StatusCreated, chat)
}

// GetChat returns a specific chat if the authenticated user is a member.
func (h *Handlers) GetChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	chatID := c.Param("id")

	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ?", chatID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this chat"})
		return
	}

	var chat models.Chat
	if err := h.db.Preload("Members.User").First(&chat, "id = ?", chatID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
		return
	}

	c.JSON(http.StatusOK, chat)
}

type UpdateChatRequest struct {
	Name          *string  `json:"name,omitempty"`
	AddMembers    []string `json:"add_members,omitempty"`
	RemoveMembers []string `json:"remove_members,omitempty"`
}

// UpdateChat updates a chat (admin only): rename, add/remove members.
func (h *Handlers) UpdateChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	chatID := c.Param("id")

	var member models.ChatMember
	if err := h.db.Where("chat_id = ? AND user_id = ? AND role = ?", chatID, userID, "admin").First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can update chat"})
		return
	}

	var req UpdateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		h.db.Model(&models.Chat{}).Where("id = ?", chatID).Update("name", *req.Name)
	}

	now := time.Now()
	for _, memberID := range req.AddMembers {
		h.db.FirstOrCreate(&models.ChatMember{
			ChatID: chatID, UserID: memberID, Role: "member", JoinedAt: now, LastReadAt: now,
		}, "chat_id = ? AND user_id = ?", chatID, memberID)
	}

	for _, memberID := range req.RemoveMembers {
		h.db.Where("chat_id = ? AND user_id = ? AND role != ?", chatID, memberID, "admin").Delete(&models.ChatMember{})
	}

	var chat models.Chat
	h.db.Preload("Members.User").First(&chat, "id = ?", chatID)
	c.JSON(http.StatusOK, chat)
}

// GetOrCreateDirectChat finds an existing direct chat between two users, or creates one.
func (h *Handlers) GetOrCreateDirectChat(userID1, userID2 string) (*models.Chat, error) {
	var chatID string
	row := h.db.Raw(`
		SELECT cm1.chat_id FROM chat_members cm1
		JOIN chat_members cm2 ON cm1.chat_id = cm2.chat_id
		JOIN chats c ON c.id = cm1.chat_id
		WHERE cm1.user_id = ? AND cm2.user_id = ? AND c.type = 'direct'
		LIMIT 1
	`, userID1, userID2).Row()

	if err := row.Scan(&chatID); err == nil {
		var chat models.Chat
		h.db.Preload("Members.User").First(&chat, "id = ?", chatID)
		return &chat, nil
	}

	chat := models.Chat{Type: "direct", CreatedBy: userID1}
	if err := h.db.Create(&chat).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	members := []models.ChatMember{
		{ChatID: chat.ID, UserID: userID1, Role: "member", JoinedAt: now, LastReadAt: now},
		{ChatID: chat.ID, UserID: userID2, Role: "member", JoinedAt: now, LastReadAt: now},
	}
	if err := h.db.Create(&members).Error; err != nil {
		return nil, err
	}

	h.db.Preload("Members.User").First(&chat, "id = ?", chat.ID)
	return &chat, nil
}
