package handlers_test

import (
	"net/http"
	"testing"

	"familycall/server/internal/models"
)

func TestCreateInvite(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	body := map[string]string{"contact_name": "FamilyMember"}
	resp := doJSON(t, env, "POST", "/api/invite", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)

	var invite models.Invite
	decodeBody(t, resp, &invite)

	if invite.ContactName != "FamilyMember" {
		t.Fatalf("expected contact_name=FamilyMember, got %s", invite.ContactName)
	}
	if invite.UUID == "" {
		t.Fatal("expected non-empty UUID")
	}
}

func TestCreateInvite_NonOrganizer(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create and register a member
	inv := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", inv, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)
	var regResult struct {
		Token string `json:"token"`
	}
	decodeBody(t, resp, &regResult)

	// Member tries to create invite — should fail
	body := map[string]string{"contact_name": "Hacker"}
	resp = doJSON(t, env, "POST", "/api/invite", body, regResult.Token)
	assertStatus(t, resp, http.StatusForbidden)
}

func TestGetInvite(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	body := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	// Get invite (public endpoint)
	resp = doRequest(t, env, "GET", "/api/invite/"+invite.UUID, "")
	assertStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	decodeBody(t, resp, &result)

	if result["contact_name"] != "Member1" {
		t.Fatalf("expected contact_name=Member1, got %v", result["contact_name"])
	}
	if result["accepted"] != false {
		t.Fatal("expected accepted=false for pending invite")
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	env := setupTestEnv(t)

	resp := doRequest(t, env, "GET", "/api/invite/nonexistent-uuid", "")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestGetPendingInvites(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create two invites
	for _, name := range []string{"Member1", "Member2"} {
		body := map[string]string{"contact_name": name}
		resp := doJSON(t, env, "POST", "/api/invite", body, orgToken)
		assertStatus(t, resp, http.StatusCreated)
	}

	// Get pending
	resp := doRequest(t, env, "GET", "/api/invites/pending", orgToken)
	assertStatus(t, resp, http.StatusOK)

	var result struct {
		PendingInvites []map[string]interface{} `json:"pending_invites"`
	}
	decodeBody(t, resp, &result)

	if len(result.PendingInvites) != 2 {
		t.Fatalf("expected 2 pending invites, got %d", len(result.PendingInvites))
	}
}

func TestDeleteInvite(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	body := map[string]string{"contact_name": "ToDelete"}
	resp := doJSON(t, env, "POST", "/api/invite", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	// Delete invite
	resp = doRequest(t, env, "DELETE", "/api/invites/"+invite.ID, orgToken)
	assertStatus(t, resp, http.StatusOK)

	// Verify it's gone
	resp = doRequest(t, env, "GET", "/api/invite/"+invite.UUID, "")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestCreateInvite_DuplicateUsername(t *testing.T) {
	env := setupTestEnv(t)
	orgToken, _ := registerOrganizer(t, env, "Organizer")

	// Create invite, register user
	body := map[string]string{"contact_name": "Member1"}
	resp := doJSON(t, env, "POST", "/api/invite", body, orgToken)
	assertStatus(t, resp, http.StatusCreated)
	var invite models.Invite
	decodeBody(t, resp, &invite)

	resp = doJSON(t, env, "POST", "/api/register", map[string]string{"username": "Member1", "invite_uuid": invite.UUID}, "")
	assertStatus(t, resp, http.StatusCreated)

	// Try to create invite with same name — should fail since user already exists
	resp = doJSON(t, env, "POST", "/api/invite", body, orgToken)
	assertStatus(t, resp, http.StatusConflict)
}
