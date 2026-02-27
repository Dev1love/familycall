package handlers_test

import (
	"net/http"
	"testing"

	"familycall/server/internal/models"
)

func TestRegistrationStatus_Empty(t *testing.T) {
	env := setupTestEnv(t)

	resp := doRequest(t, env, "GET", "/api/registration-status", "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	decodeBody(t, resp, &result)

	if result["registration_enabled"] != true {
		t.Fatal("expected registration_enabled=true when no users exist")
	}
}

func TestRegisterFirstUser(t *testing.T) {
	env := setupTestEnv(t)

	token, userID := registerOrganizer(t, env, "Organizer")

	if token == "" || userID == "" {
		t.Fatal("expected non-empty token and userID")
	}

	// Registration should now be disabled
	resp := doRequest(t, env, "GET", "/api/registration-status", "")
	assertStatus(t, resp, http.StatusOK)

	var status map[string]interface{}
	decodeBody(t, resp, &status)

	if status["registration_enabled"] != false {
		t.Fatal("expected registration_enabled=false after first user registered")
	}
}

func TestRegisterWithoutInvite_Forbidden(t *testing.T) {
	env := setupTestEnv(t)
	registerOrganizer(t, env, "Organizer")

	// Try to register without invite
	body := map[string]string{"username": "NewUser"}
	resp := doJSON(t, env, "POST", "/api/register", body, "")
	assertStatus(t, resp, http.StatusForbidden)
}

func TestRegisterWithInvite(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create invite
	inviteBody := map[string]string{"contact_name": "FamilyMember"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var invite models.Invite
	decodeBody(t, resp, &invite)

	// Register with invite
	regBody := map[string]string{
		"username":    "FamilyMember",
		"invite_uuid": invite.UUID,
	}
	resp = doJSON(t, env, "POST", "/api/register", regBody, "")
	assertStatus(t, resp, http.StatusCreated)

	var result struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	decodeBody(t, resp, &result)

	if result.User.Username != "FamilyMember" {
		t.Fatalf("expected username FamilyMember, got %s", result.User.Username)
	}
	if result.User.InvitedByUserID == nil {
		t.Fatal("expected InvitedByUserID to be set")
	}
}

func TestRegisterWithInvalidInvite(t *testing.T) {
	env := setupTestEnv(t)
	registerOrganizer(t, env, "Organizer")

	body := map[string]string{
		"username":    "Hacker",
		"invite_uuid": "non-existent-uuid",
	}
	resp := doJSON(t, env, "POST", "/api/register", body, "")
	assertStatus(t, resp, http.StatusForbidden)
}

func TestRegisterDuplicateUsername(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create invite and register
	inviteBody := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inviteBody, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var invite models.Invite
	decodeBody(t, resp, &invite)

	regBody := map[string]string{"username": "Member1", "invite_uuid": invite.UUID}
	resp = doJSON(t, env, "POST", "/api/register", regBody, "")
	assertStatus(t, resp, http.StatusCreated)

	// Try to register again with same username
	resp = doJSON(t, env, "POST", "/api/register", regBody, "")
	assertStatus(t, resp, http.StatusConflict)
}

func TestLogin(t *testing.T) {
	env := setupTestEnv(t)
	registerOrganizer(t, env, "Organizer")

	body := map[string]string{"username": "Organizer"}
	resp := doJSON(t, env, "POST", "/api/login", body, "")
	assertStatus(t, resp, http.StatusOK)

	var result struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	decodeBody(t, resp, &result)

	if result.Token == "" {
		t.Fatal("expected non-empty token on login")
	}
	if result.User.Username != "Organizer" {
		t.Fatalf("expected Organizer, got %s", result.User.Username)
	}
}

func TestLoginInvalidUsername(t *testing.T) {
	env := setupTestEnv(t)
	registerOrganizer(t, env, "Organizer")

	body := map[string]string{"username": "Nobody"}
	resp := doJSON(t, env, "POST", "/api/login", body, "")
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestGetMe(t *testing.T) {
	env := setupTestEnv(t)
	token, _ := registerOrganizer(t, env, "Organizer")

	resp := doRequest(t, env, "GET", "/api/me", token)
	assertStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	decodeBody(t, resp, &result)

	if result["username"] != "Organizer" {
		t.Fatalf("expected Organizer, got %v", result["username"])
	}
	if result["is_first_user"] != true {
		t.Fatal("expected is_first_user=true for organizer")
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	env := setupTestEnv(t)

	resp := doRequest(t, env, "GET", "/api/me", "")
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestGetMe_InvalidToken(t *testing.T) {
	env := setupTestEnv(t)

	resp := doRequest(t, env, "GET", "/api/me", "invalid-token")
	assertStatus(t, resp, http.StatusUnauthorized)
}

func TestRegisterUsernameValidation(t *testing.T) {
	env := setupTestEnv(t)

	// Username too short (min=3)
	body := map[string]string{"username": "ab"}
	resp := doJSON(t, env, "POST", "/api/register", body, "")
	assertStatus(t, resp, http.StatusBadRequest)

	// Empty username
	body = map[string]string{"username": ""}
	resp = doJSON(t, env, "POST", "/api/register", body, "")
	assertStatus(t, resp, http.StatusBadRequest)
}
