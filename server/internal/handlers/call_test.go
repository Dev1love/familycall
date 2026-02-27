package handlers_test

import (
	"net/http"
	"testing"

	"familycall/server/internal/models"
)

// setupTwoUsers creates organizer + member, returns both tokens and IDs.
func setupTwoUsers(t *testing.T, env *testEnv) (orgToken, orgID, memberToken, memberID string) {
	t.Helper()
	orgToken, orgID = registerOrganizer(t, env, "Organizer")

	// Create invite and register member
	resp := doJSON(t, env, "POST", "/api/invite", map[string]string{"contact_name": "Member1"}, orgToken)
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
	memberToken = regResult.Token
	memberID = regResult.User.ID
	return
}

// --- P2P Call Tests ---

func TestInitiateCall_Video(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, _, memberID := setupTwoUsers(t, env)

	body := map[string]string{
		"contact_id": memberID,
		"call_type":  "video",
	}
	resp := doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	decodeBody(t, resp, &result)
	if result["call_type"] != "video" {
		t.Fatalf("expected call_type=video, got %v", result["call_type"])
	}
}

func TestInitiateCall_Audio(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, _, memberID := setupTwoUsers(t, env)

	body := map[string]string{
		"contact_id": memberID,
		"call_type":  "audio",
	}
	resp := doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	decodeBody(t, resp, &result)
	if result["call_type"] != "audio" {
		t.Fatalf("expected call_type=audio, got %v", result["call_type"])
	}
}

func TestInitiateCall_InvalidCallType(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, _, memberID := setupTwoUsers(t, env)

	body := map[string]string{
		"contact_id": memberID,
		"call_type":  "invalid",
	}
	resp := doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestInitiateCall_CallYourself(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, orgID, _, _ := setupTwoUsers(t, env)

	body := map[string]string{
		"contact_id": orgID,
		"call_type":  "video",
	}
	resp := doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)

	var result map[string]string
	decodeBody(t, resp, &result)
	if result["error"] != "Cannot call yourself" {
		t.Fatalf("expected 'Cannot call yourself', got %s", result["error"])
	}
}

func TestInitiateCall_NonExistentUser(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, _, _ := setupTwoUsers(t, env)

	body := map[string]string{
		"contact_id": "non-existent-id",
		"call_type":  "video",
	}
	resp := doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestInitiateCall_MissingFields(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, _, _ := setupTwoUsers(t, env)

	// Missing contact_id
	body := map[string]string{"call_type": "video"}
	resp := doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)

	// Missing call_type
	body = map[string]string{"contact_id": "some-id"}
	resp = doJSON(t, env, "POST", "/api/call", body, orgToken)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestInitiateCall_Unauthorized(t *testing.T) {
	env := setupTestEnv(t)

	body := map[string]string{
		"contact_id": "some-id",
		"call_type":  "video",
	}
	resp := doJSON(t, env, "POST", "/api/call", body, "")
	assertStatus(t, resp, http.StatusUnauthorized)
}

// --- Group Call Tests ---

func TestStartGroupCall(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var call models.GroupCall
	decodeBody(t, resp, &call)

	if call.ChatID != chatID {
		t.Fatalf("expected chat_id=%s, got %s", chatID, call.ChatID)
	}
	if call.EndedAt != nil {
		t.Fatal("expected ended_at to be nil for active call")
	}
	if len(call.Participants) != 1 {
		t.Fatalf("expected 1 participant (starter), got %d", len(call.Participants))
	}
}

func TestStartGroupCall_DuplicateActiveCall(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Start first call
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	// Try to start second call — should conflict
	resp = doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusConflict)
}

func TestStartGroupCall_NotMember(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Create a third user not in the chat
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

	// Outsider tries to start call — forbidden
	resp = doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, regResult.Token)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestJoinGroupCall(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	// Start call
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call models.GroupCall
	decodeBody(t, resp, &call)

	// Member joins
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/join", nil, memberToken)
	assertStatus(t, resp, http.StatusOK)

	var joinResult struct {
		CallID       string                       `json:"call_id"`
		Participants []models.GroupCallParticipant `json:"participants"`
	}
	decodeBody(t, resp, &joinResult)

	if joinResult.CallID != call.ID {
		t.Fatalf("expected call_id=%s, got %s", call.ID, joinResult.CallID)
	}
	// participants list shows active participants OTHER than the joiner
	if len(joinResult.Participants) != 1 {
		t.Fatalf("expected 1 other participant (organizer), got %d", len(joinResult.Participants))
	}
}

func TestJoinGroupCall_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	_, memberToken, _ := setupChatEnv(t, env)

	resp := doJSON(t, env, "POST", "/api/calls/non-existent-id/join", nil, memberToken)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestLeaveGroupCall(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	// Start call
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call models.GroupCall
	decodeBody(t, resp, &call)

	// Member joins
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/join", nil, memberToken)
	assertStatus(t, resp, http.StatusOK)

	// Member leaves
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/leave", nil, memberToken)
	assertStatus(t, resp, http.StatusOK)

	var leaveResult struct {
		Left      bool  `json:"left"`
		Remaining int64 `json:"remaining"`
	}
	decodeBody(t, resp, &leaveResult)

	if !leaveResult.Left {
		t.Fatal("expected left=true")
	}
	if leaveResult.Remaining != 1 {
		t.Fatalf("expected 1 remaining (organizer), got %d", leaveResult.Remaining)
	}
}

func TestLeaveGroupCall_AutoEnd(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Start call (only organizer)
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call models.GroupCall
	decodeBody(t, resp, &call)

	// Organizer leaves — call should auto-end (0 remaining)
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/leave", nil, orgToken)
	assertStatus(t, resp, http.StatusOK)

	var leaveResult struct {
		Left      bool  `json:"left"`
		Remaining int64 `json:"remaining"`
	}
	decodeBody(t, resp, &leaveResult)

	if leaveResult.Remaining != 0 {
		t.Fatalf("expected 0 remaining, got %d", leaveResult.Remaining)
	}

	// Verify call ended_at is set in DB
	var endedCall models.GroupCall
	env.DB.First(&endedCall, "id = ?", call.ID)
	if endedCall.EndedAt == nil {
		t.Fatal("expected call to be ended (ended_at set)")
	}
}

func TestLeaveGroupCall_NotParticipant(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	// Start call (only organizer joins)
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call models.GroupCall
	decodeBody(t, resp, &call)

	// Member tries to leave without joining — not found
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/leave", nil, memberToken)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestJoinGroupCall_EndedCall(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	// Start call
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call models.GroupCall
	decodeBody(t, resp, &call)

	// End the call (organizer leaves, 0 remaining)
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/leave", nil, orgToken)
	assertStatus(t, resp, http.StatusOK)

	// Member tries to join ended call
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/join", nil, memberToken)
	assertStatus(t, resp, http.StatusGone)
}

func TestRejoinGroupCall(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, memberToken, chatID := setupChatEnv(t, env)

	// Start call
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call models.GroupCall
	decodeBody(t, resp, &call)

	// Member joins
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/join", nil, memberToken)
	assertStatus(t, resp, http.StatusOK)

	// Member leaves
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/leave", nil, memberToken)
	assertStatus(t, resp, http.StatusOK)

	// Member rejoins
	resp = doJSON(t, env, "POST", "/api/calls/"+call.ID+"/join", nil, memberToken)
	assertStatus(t, resp, http.StatusOK)

	var joinResult struct {
		CallID string `json:"call_id"`
	}
	decodeBody(t, resp, &joinResult)
	if joinResult.CallID != call.ID {
		t.Fatalf("expected call_id=%s on rejoin, got %s", call.ID, joinResult.CallID)
	}
}

func TestStartNewCallAfterEnd(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _, chatID := setupChatEnv(t, env)

	// Start and end first call
	resp := doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var call1 models.GroupCall
	decodeBody(t, resp, &call1)

	resp = doJSON(t, env, "POST", "/api/calls/"+call1.ID+"/leave", nil, orgToken)
	assertStatus(t, resp, http.StatusOK)

	// Start second call — should work since first ended
	resp = doJSON(t, env, "POST", "/api/chats/"+chatID+"/calls", nil, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var call2 models.GroupCall
	decodeBody(t, resp, &call2)

	if call2.ID == call1.ID {
		t.Fatal("expected new call ID after ending previous")
	}
}
