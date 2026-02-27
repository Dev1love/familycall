package handlers_test

import (
	"net/http"
	"testing"

	"familycall/server/internal/models"
)

func TestCreateChat(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, orgID := registerOrganizer(t, env, "Organizer")

	// Create invite and register second user
	inviteBody := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	regBody := map[string]string{"username": "Member1", "invite_uuid": invite.UUID}
	resp = doJSON(t, env, "POST", "/api/register", regBody, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		User models.User `json:"user"`
	}
	decodeBody(t, resp, &regResult)
	memberID := regResult.User.ID

	// Create group chat
	chatBody := map[string]interface{}{
		"name":       "Family Chat",
		"member_ids": []string{memberID},
	}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var chat models.Chat
	decodeBody(t, resp, &chat)

	if chat.Name == nil || *chat.Name != "Family Chat" {
		t.Fatal("expected chat name 'Family Chat'")
	}
	if chat.CreatedBy != orgID {
		t.Fatalf("expected created_by=%s, got %s", orgID, chat.CreatedBy)
	}
	if len(chat.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(chat.Members))
	}
}

func TestListChats(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create invite and register second user
	inviteBody := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	regBody := map[string]string{"username": "Member1", "invite_uuid": invite.UUID}
	resp = doJSON(t, env, "POST", "/api/register", regBody, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		User models.User `json:"user"`
	}
	decodeBody(t, resp, &regResult)

	// Create chat
	chatBody := map[string]interface{}{
		"name":       "Test Chat",
		"member_ids": []string{regResult.User.ID},
	}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	// List chats
	resp = doRequest(t, env, "GET", "/api/chats", orgToken)
	assertStatus(t, resp, http.StatusOK)

	var chats []map[string]interface{}
	decodeBody(t, resp, &chats)

	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
}

func TestListChats_Empty(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	resp := doRequest(t, env, "GET", "/api/chats", orgToken)
	assertStatus(t, resp, http.StatusOK)

	var chats []map[string]interface{}
	decodeBody(t, resp, &chats)

	if len(chats) != 0 {
		t.Fatalf("expected 0 chats, got %d", len(chats))
	}
}

func TestGetChat(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create second user
	inviteBody := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		User  models.User `json:"user"`
		Token string      `json:"token"`
	}
	decodeBody(t, resp, &regResult)

	// Create chat
	chatBody := map[string]interface{}{"name": "Test Chat", "member_ids": []string{regResult.User.ID}}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var chat models.Chat
	decodeBody(t, resp, &chat)

	// Get chat as organizer
	resp = doRequest(t, env, "GET", "/api/chats/"+chat.ID, orgToken)
	assertStatus(t, resp, http.StatusOK)

	// Get chat as member
	resp = doRequest(t, env, "GET", "/api/chats/"+chat.ID, regResult.Token)
	assertStatus(t, resp, http.StatusOK)
}

func TestGetChat_NotMember(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create two more users
	inv1 := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inv1, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite1 models.Invite
	decodeBody(t, resp, &invite1)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite1.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var reg1 struct {
		User models.User `json:"user"`
	}
	decodeBody(t, resp, &reg1)

	inv2 := map[string]string{"contact_name": "Member2"}
	resp = doJSON(t, env, "POST", "/api/invite", inv2, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite2 models.Invite
	decodeBody(t, resp, &invite2)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member2", "invite_uuid": invite2.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var reg2 struct {
		User  models.User `json:"user"`
		Token string      `json:"token"`
	}
	decodeBody(t, resp, &reg2)

	// Create chat with only Member1
	chatBody := map[string]interface{}{"name": "Private Chat", "member_ids": []string{reg1.User.ID}}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var chat models.Chat
	decodeBody(t, resp, &chat)

	// Member2 should not be able to see the chat
	resp = doRequest(t, env, "GET", "/api/chats/"+chat.ID, reg2.Token)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestUpdateChat(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create second user
	inviteBody := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		User models.User `json:"user"`
	}
	decodeBody(t, resp, &regResult)

	// Create chat
	chatBody := map[string]interface{}{"name": "Old Name", "member_ids": []string{regResult.User.ID}}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var chat models.Chat
	decodeBody(t, resp, &chat)

	// Rename chat
	newName := "New Name"
	updateBody := map[string]interface{}{"name": newName}
	resp = doJSON(t, env, "PUT", "/api/chats/"+chat.ID, updateBody, orgToken)
	assertStatus(t, resp, http.StatusOK)

	var updated models.Chat
	decodeBody(t, resp, &updated)

	if updated.Name == nil || *updated.Name != "New Name" {
		t.Fatalf("expected name 'New Name', got %v", updated.Name)
	}
}

func TestUpdateChat_NonAdmin(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create second user
	inviteBody := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		User  models.User `json:"user"`
		Token string      `json:"token"`
	}
	decodeBody(t, resp, &regResult)

	// Create chat
	chatBody := map[string]interface{}{"name": "Chat", "member_ids": []string{regResult.User.ID}}
	resp = doJSON(t, env, "POST", "/api/chats", chatBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var chat models.Chat
	decodeBody(t, resp, &chat)

	// Member tries to update — should be forbidden
	updateBody := map[string]interface{}{"name": "Hacked"}
	resp = doJSON(t, env, "PUT", "/api/chats/"+chat.ID, updateBody, regResult.Token)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestCreateChat_Validation(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Missing name
	body := map[string]interface{}{"member_ids": []string{"some-id"}}
	resp := doJSON(t, env, "POST", "/api/chats", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)

	// Missing members
	body = map[string]interface{}{"name": "Chat"}
	resp = doJSON(t, env, "POST", "/api/chats", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)
}
