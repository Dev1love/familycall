package handlers_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"familycall/server/internal/config"
	"familycall/server/internal/handlers"
	"familycall/server/internal/models"
	"familycall/server/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// testEnv holds all dependencies for handler tests.
type testEnv struct {
	DB     *gorm.DB
	Hub    *websocket.Hub
	Config *config.Config
	H      *handlers.Handlers
	Router *gin.Engine
}

// setupTestEnv creates an in-memory SQLite database and wires up handlers.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Use a unique shared-cache in-memory DB per test to avoid data leaks between tests.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Invite{},
		&models.PushSubscription{},
		&models.Chat{},
		&models.ChatMember{},
		&models.Message{},
		&models.GroupCall{},
		&models.GroupCallParticipant{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}

	hub := websocket.NewHub()
	go hub.Run()

	cfg := &config.Config{
		JWTSecret:    "test-secret-key-for-testing",
		DatabasePath: ":memory:",
		TURNPort:     3478,
		TURNRealm:    "test",
		BackendOnly:  true,
		BackendPort:  "8080",
		FrontendURI:  "http://localhost:3000",
		VAPIDKeys: &config.VAPIDKeys{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
			Subject:    "mailto:test@test.com",
		},
	}

	// Create an empty embed.FS (translations won't be tested here)
	var emptyFS embed.FS

	h := handlers.New(db, hub, cfg, nil, emptyFS)

	router := gin.New()

	// Public routes
	api := router.Group("/api")
	api.POST("/register", h.Register)
	api.POST("/login", h.Login)
	api.GET("/registration-status", h.CheckRegistrationStatus)
	api.GET("/invite/:uuid", h.GetInvite)

	// Protected routes
	protected := api.Group("")
	protected.Use(h.AuthMiddleware())
	{
		protected.GET("/me", h.GetMe)
		protected.POST("/invite", h.CreateInvite)
		protected.GET("/invites/pending", h.GetPendingInvites)
		protected.DELETE("/invites/:id", h.DeleteInvite)
		protected.POST("/invite/:uuid/accept", h.AcceptInvite)
		protected.GET("/chats", h.ListChats)
		protected.POST("/chats", h.CreateChat)
		protected.GET("/chats/:id", h.GetChat)
		protected.PUT("/chats/:id", h.UpdateChat)
		protected.GET("/chats/:id/messages", h.ListMessages)
		protected.POST("/chats/:id/messages", h.SendMessage)
		protected.PUT("/messages/:id", h.EditMessage)
		protected.DELETE("/messages/:id", h.DeleteMessage)

		// Call routes
		protected.POST("/call", h.InitiateCall)
		protected.POST("/chats/:id/calls", h.StartGroupCall)
		protected.POST("/calls/:id/join", h.JoinGroupCall)
		protected.POST("/calls/:id/leave", h.LeaveGroupCall)
	}

	// WebSocket route
	router.GET("/ws", h.HandleWebSocket)

	return &testEnv{
		DB:     db,
		Hub:    hub,
		Config: cfg,
		H:      h,
		Router: router,
	}
}

// helper: register the first user (organizer) and return token + user ID.
func registerOrganizer(t *testing.T, env *testEnv, username string) (token, userID string) {
	t.Helper()
	body := map[string]string{"username": username}
	resp := doJSON(t, env, "POST", "/api/register", body, "")
	assertStatus(t, resp, http.StatusCreated)

	var result struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	decodeBody(t, resp, &result)

	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
	return result.Token, result.User.ID
}

// helper: do a JSON request.
func doJSON(t *testing.T, env *testEnv, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	return w
}

func doRequest(t *testing.T, env *testEnv, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	return w
}

func assertStatus(t *testing.T, resp *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if resp.Code != expected {
		t.Fatalf("expected status %d, got %d; body: %s", expected, resp.Code, resp.Body.String())
	}
}

func decodeBody(t *testing.T, resp *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

// Ensure embed.FS satisfies fs.FS at compile time.
var _ fs.FS = embed.FS{}
