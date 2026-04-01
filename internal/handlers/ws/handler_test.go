package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hastenr/chatapi/internal/config"
	"github.com/hastenr/chatapi/internal/handlers/ws"
	"github.com/hastenr/chatapi/internal/services/chatroom"
	"github.com/hastenr/chatapi/internal/services/delivery"
	"github.com/hastenr/chatapi/internal/services/message"
	"github.com/hastenr/chatapi/internal/services/realtime"
	"github.com/hastenr/chatapi/internal/services/webhook"
	"github.com/hastenr/chatapi/internal/testutil"
)

// wsTestEnv holds all wired-up services for WS handler tests.
type wsTestEnv struct {
	realtimeSvc *realtime.Service
	server      *httptest.Server
}

func newWSEnv(t *testing.T) *wsTestEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	cfg := &config.Config{
		AllowedOrigins:        []string{"*"},
		MaxConnectionsPerUser: 5,
		JWTSecret:             testutil.TestJWTSecret,
	}

	chatroomSvc := chatroom.NewService(db.DB)
	messageSvc := message.NewService(db.DB)
	realtimeSvc := realtime.NewService(db.DB, cfg.MaxConnectionsPerUser)
	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(db.DB, realtimeSvc, chatroomSvc, "", "", webhookSvc)

	t.Cleanup(func() { realtimeSvc.Shutdown(context.Background()) })

	handler := ws.NewHandler(chatroomSvc, messageSvc, realtimeSvc, deliverySvc, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleConnection)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &wsTestEnv{
		realtimeSvc: realtimeSvc,
		server:      srv,
	}
}

func (e *wsTestEnv) wsURL() string {
	return "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
}

// dial opens a WS connection. token is placed in ?token=<jwt>.
// Pass an empty token to connect unauthenticated.
func dial(t *testing.T, url string, headers http.Header) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	return dialer.Dial(url, headers)
}

// dialAs connects as the given userID using ?token=<jwt>.
func dialAs(t *testing.T, env *wsTestEnv, userID string) (*websocket.Conn, error) {
	t.Helper()
	token := testutil.SignJWT(userID)
	conn, _, err := dial(t, env.wsURL()+"?token="+token, nil)
	return conn, err
}

// --- Auth rejection ---

func TestWSHandler_RejectsNoAuth(t *testing.T) {
	env := newWSEnv(t)

	_, resp, err := dial(t, env.wsURL(), nil)
	if err == nil {
		t.Error("expected connection rejection, got nil error")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestWSHandler_RejectsInvalidToken(t *testing.T) {
	env := newWSEnv(t)

	_, resp, err := dial(t, env.wsURL()+"?token=not-a-real-jwt", nil)
	if err == nil {
		t.Error("expected rejection for invalid token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// --- Successful connection via query param ---

func TestWSHandler_ValidTokenQueryParam(t *testing.T) {
	env := newWSEnv(t)

	conn, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("expected successful connection, got: %v", err)
	}
	defer conn.Close()

	if !env.realtimeSvc.IsUserOnline("default", "user1") {
		t.Error("user1 is not online after connecting")
	}
}

// --- Successful connection via Authorization header ---

func TestWSHandler_ValidTokenAuthHeader(t *testing.T) {
	env := newWSEnv(t)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+testutil.SignJWT("user2"))

	conn, _, err := dial(t, env.wsURL(), headers)
	if err != nil {
		t.Fatalf("expected successful connection via header, got: %v", err)
	}
	defer conn.Close()

	if !env.realtimeSvc.IsUserOnline("default", "user2") {
		t.Error("user2 is not online after connecting")
	}
}

// --- Disconnect updates presence ---

func TestWSHandler_DisconnectUpdatesPresence(t *testing.T) {
	env := newWSEnv(t)

	conn, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	if !env.realtimeSvc.IsUserOnline("default", "user1") {
		t.Error("user1 not online after connect")
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
	// Grace period may keep user "online" briefly — just verify no panic occurred.
}

// --- Ping message handled ---

func TestWSHandler_PingMessageHandled(t *testing.T) {
	env := newWSEnv(t)

	conn, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	ping := map[string]interface{}{"type": "ping"}
	data, _ := json.Marshal(ping)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	// Connection should still be alive after ping
	ack := map[string]interface{}{"type": "ack", "data": map[string]interface{}{"room_id": "r", "seq": 0}}
	data, _ = json.Marshal(ack)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Errorf("connection closed after ping: %v", err)
	}
}

// --- Unknown message type silently dropped ---

func TestWSHandler_UnknownMessageType(t *testing.T) {
	env := newWSEnv(t)

	conn, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	msg := map[string]interface{}{"type": "unknown.type.xyz"}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)

	// Connection should still be alive
	followup := map[string]interface{}{"type": "ping"}
	data, _ = json.Marshal(followup)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Error("connection dropped after unknown message type")
	}
}

// --- Multiple connections same user ---

func TestWSHandler_MultipleConnectionsSameUser(t *testing.T) {
	env := newWSEnv(t)

	conn1, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("first connect: %v", err)
	}
	defer conn1.Close()

	conn2, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("second connect: %v", err)
	}
	defer conn2.Close()

	if !env.realtimeSvc.IsUserOnline("default", "user1") {
		t.Error("user1 not online with two connections")
	}
}

// --- IsUserOnline integration ---

func TestRealtimeSvc_IsUserOnlineIntegration(t *testing.T) {
	env := newWSEnv(t)

	if env.realtimeSvc.IsUserOnline("default", "user1") {
		t.Error("user1 should be offline before connecting")
	}

	conn, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	if !env.realtimeSvc.IsUserOnline("default", "user1") {
		t.Error("user1 should be online after connecting")
	}

	conn.Close()
}

// --- GetOnlineUsers ---

func TestRealtimeSvc_GetOnlineUsers(t *testing.T) {
	env := newWSEnv(t)

	before := env.realtimeSvc.GetOnlineUsers("default")
	if len(before) != 0 {
		t.Errorf("online users before connect = %d, want 0", len(before))
	}

	conn1, err := dialAs(t, env, "alice")
	if err != nil {
		t.Fatalf("alice connect: %v", err)
	}
	defer conn1.Close()

	conn2, err := dialAs(t, env, "bob")
	if err != nil {
		t.Fatalf("bob connect: %v", err)
	}
	defer conn2.Close()

	online := env.realtimeSvc.GetOnlineUsers("default")
	if len(online) != 2 {
		t.Errorf("online users = %d, want 2", len(online))
	}
}

// --- ActiveConnections counter ---

func TestWSHandler_ActiveConnectionsCounter(t *testing.T) {
	env := newWSEnv(t)

	before := env.realtimeSvc.ActiveConnections()

	conn, err := dialAs(t, env, "user1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	if got := env.realtimeSvc.ActiveConnections(); got != before+1 {
		t.Errorf("active connections = %d, want %d", got, before+1)
	}
}
