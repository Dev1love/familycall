package handlers_test

import (
	"net/http"
	"testing"

	"familycall/server/internal/models"
)

// helper: create organizer + member + chat, return tokens and chat ID.
func setupChatEnv(t *testing.T, env *testEnv) (orgToken, memberToken, chatID string) {
	t.Helper()

	orgToken, _ = registerOrganizer(t, env, "Organizer")

	// Create invite
	resp := doJSON(t, env, "POST", "/api/invite", map[string]string{"contact_name": "Member1"}, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	// Register member
	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		User  models.User `json:"user"`
		Token string      `json:"token"`
	}
	decodeBody(t, resp, &regResult)
	memberToken = regResult.Token

	// Create chat
	chatBody := map[string]interface{}{"name": "Test Chat", "member_ids": []string{regResult.User.ID}}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var chat models.Chat
	decodeBody(t, resp, &chat)
	chatID = chat.ID

	return
}

func TestSendMessage(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	body := map[string]string{"content": "Hello family!"}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var msg models.Message
	decodeBody(t, resp, &msg)

	if msg.Content != "Hello family!" {
		t.Fatalf("expected 'Hello family!', got %s", msg.Content)
	}
	if msg.ChatID != chatID {
		t.Fatalf("expected chat_id=%s, got %s", chatID, msg.ChatID)
	}
}

func TestSendMessage_XSSEscaping(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	body := map[string]string{"content": "<script>alert('xss')</script>"}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var msg models.Message
	decodeBody(t, resp, &msg)

	if msg.Content == "<script>alert('xss')</script>" {
		t.Fatal("expected HTML to be escaped")
	}
}

func TestSendMessage_NotMember(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Create a third user who is NOT in the chat
	resp := doJSON(t, env, "POST", "/api/invite", map[string]string{"contact_name": "Outsider"}, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Outsider", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		Token string `json:"token"`
	}
	decodeBody(t, resp, &regResult)

	body := map[string]string{"content": "I shouldn't be here"}
	resp = doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, regResult.Token)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestListMessages(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Send a few messages
	for i := 0; i < 3; i++ {
		body := map[string]string{"content": "Message"}
		resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
		assertStatus(t, resp, http.StatusCreated)
	}

	// List messages
	resp := doRequest(t, env, "GET", "/api/chats/"+chatID+"/messages", orgToken)
	assertStatus(t, resp, http.StatusOK)

	var messages []models.Message
	decodeBody(t, resp, &messages)

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
}

func TestListMessages_Pagination(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Send 5 messages
	for i := 0; i < 5; i++ {
		body := map[string]string{"content": "Message"}
		resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
		assertStatus(t, resp, http.StatusCreated)
	}

	// Fetch with limit=2
	resp := doRequest(t, env, "GET", "/api/chats/"+chatID+"/messages?limit=2", orgToken)
	assertStatus(t, resp, http.StatusOK)

	var messages []models.Message
	decodeBody(t, resp, &messages)

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages with limit=2, got %d", len(messages))
	}
}

func TestEditMessage(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Send message
	body := map[string]string{"content": "Original"}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var msg models.Message
	decodeBody(t, resp, &msg)

	// Edit message
	editBody := map[string]string{"content": "Edited"}
	resp = doJSON(t, env, "PUT", "/api/messages/"+msg.ID, editBody, orgToken)
	assertStatus(t, resp, http.StatusOK)

	var edited models.Message
	decodeBody(t, resp, &edited)

	if edited.Content != "Edited" {
		t.Fatalf("expected 'Edited', got %s", edited.Content)
	}
	if edited.EditedAt == nil {
		t.Fatal("expected edited_at to be set")
	}
}

func TestEditMessage_NotOwner(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	// Organizer sends message
	body := map[string]string{"content": "Original"}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var msg models.Message
	decodeBody(t, resp, &msg)

	// Member tries to edit — forbidden
	editBody := map[string]string{"content": "Hacked"}
	resp = doJSON(t, env, "PUT", "/api/messages/"+msg.ID, editBody, memberToken)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestDeleteMessage(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Send message
	body := map[string]string{"content": "To delete"}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var msg models.Message
	decodeBody(t, resp, &msg)

	// Delete message
	resp = doRequest(t, env, "DELETE", "/api/messages/"+msg.ID, orgToken)
	assertStatus(t, resp, http.StatusOK)

	// Verify it's gone
	resp = doRequest(t, env, "GET", "/api/chats/"+chatID+"/messages", orgToken)
	assertStatus(t, resp, http.StatusOK)
	var messages []models.Message
	decodeBody(t, resp, &messages)

	if len(messages) != 0 {
		t.Fatalf("expected 0 messages after delete, got %d", len(messages))
	}
}

func TestDeleteMessage_NotOwner(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	body := map[string]string{"content": "Can't touch this"}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var msg models.Message
	decodeBody(t, resp, &msg)

	resp = doRequest(t, env, "DELETE", "/api/messages/"+msg.ID, memberToken)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestSendMessage_EmptyContent(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	body := map[string]string{"content": ""}
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/messages", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)
}
