package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hastenr/chatapi/internal/config"
	"github.com/hastenr/chatapi/internal/handlers/rest"
	"github.com/hastenr/chatapi/internal/services/chatroom"
	"github.com/hastenr/chatapi/internal/services/delivery"
	"github.com/hastenr/chatapi/internal/services/message"
	"github.com/hastenr/chatapi/internal/services/notification"
	"github.com/hastenr/chatapi/internal/services/realtime"
	"github.com/hastenr/chatapi/internal/services/webhook"
	"github.com/hastenr/chatapi/internal/testutil"
)

// newTestHandler returns a REST handler wired to an in-memory SQLite database.
func newTestHandler(t *testing.T) *rest.Handler {
	t.Helper()

	db := testutil.NewTestDB(t)
	cfg := &config.Config{
		JWTSecret:             testutil.TestJWTSecret,
		MaxConnectionsPerUser: 5,
	}

	chatroomSvc := chatroom.NewService(db.DB)
	messageSvc := message.NewService(db.DB)
	realtimeSvc := realtime.NewService(db.DB, 5)
	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(db.DB, realtimeSvc, chatroomSvc, "", "", webhookSvc)
	notifSvc := notification.NewService(db.DB)

	t.Cleanup(func() { realtimeSvc.Shutdown(context.Background()) })

	return rest.NewHandler(chatroomSvc, messageSvc, realtimeSvc, deliverySvc, notifSvc, cfg)
}

// newMux wires all handler routes with auth middleware so PathValue and JWT
// validation work correctly in tests, matching the transport/server.go wiring.
func newMux(h *rest.Handler) *http.ServeMux {
	a := h.AuthMiddleware
	mux := http.NewServeMux()
	mux.HandleFunc("GET /rooms", a(h.HandleGetUserRooms))
	mux.HandleFunc("POST /rooms", a(h.HandleCreateRoom))
	mux.HandleFunc("GET /rooms/{room_id}", a(h.HandleGetRoom))
	mux.HandleFunc("PATCH /rooms/{room_id}", a(h.HandleUpdateRoom))
	mux.HandleFunc("GET /rooms/{room_id}/members", a(h.HandleGetRoomMembers))
	mux.HandleFunc("POST /rooms/{room_id}/messages", a(h.HandleSendMessage))
	mux.HandleFunc("GET /rooms/{room_id}/messages", a(h.HandleGetMessages))
	mux.HandleFunc("DELETE /rooms/{room_id}/messages/{message_id}", a(h.HandleDeleteMessage))
	mux.HandleFunc("PUT /rooms/{room_id}/messages/{message_id}", a(h.HandleEditMessage))
	mux.HandleFunc("POST /acks", a(h.HandleAck))
	mux.HandleFunc("POST /notify", a(h.HandleNotify))
	mux.HandleFunc("POST /subscriptions", a(h.HandleSubscribe))
	mux.HandleFunc("GET /subscriptions", a(h.HandleListSubscriptions))
	mux.HandleFunc("DELETE /subscriptions/{id}", a(h.HandleUnsubscribe))
	mux.HandleFunc("GET /admin/dead-letters", a(h.HandleGetDeadLetters))
	return mux
}

// authedReq builds a request with a Bearer JWT for userID and the right content-type.
func authedReq(method, path, body, userID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, jsonBody(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if userID != "" {
		req.Header.Set("Authorization", "Bearer "+testutil.SignJWT(userID))
	}
	return req
}

// createRoom creates a room via the mux and returns its ID.
func createRoom(t *testing.T, mux *http.ServeMux, userID, body string) string {
	t.Helper()
	req := authedReq(http.MethodPost, "/rooms", body, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("createRoom status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["room_id"].(string)
}

// sendMessage sends a message via the mux and returns its ID.
func sendMessage(t *testing.T, mux *http.ServeMux, userID, roomID, content string) string {
	t.Helper()
	req := authedReq(http.MethodPost, "/rooms/"+roomID+"/messages",
		`{"content":"`+content+`"}`, userID)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sendMessage status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp["message_id"].(string)
}

// --- Health ---

func TestHandleHealth(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

// --- Auth middleware ---

func TestAuthMiddleware_MissingToken(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	w := httptest.NewRecorder()
	h.AuthMiddleware(nopHandler)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	w := httptest.NewRecorder()
	h.AuthMiddleware(nopHandler)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.SignJWT("user1"))
	w := httptest.NewRecorder()

	called := false
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})(w, req)

	if !called {
		t.Error("inner handler not called — valid token was rejected")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Room creation ---

func TestHandleCreateRoom_GroupRoom(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"general","members":["user1","user2"]}`)
	if roomID == "" {
		t.Error("room_id is empty")
	}
}

func TestHandleCreateRoom_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	// No Authorization header
	req := httptest.NewRequest(http.MethodPost, "/rooms", jsonBody(`{"type":"group","name":"general"}`))
	w := httptest.NewRecorder()
	h.AuthMiddleware(h.HandleCreateRoom)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- Message sending ---

func TestHandleSendMessage(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"general","members":["user1","user2"]}`)
	msgID := sendMessage(t, mux, "user1", roomID, "hello world")
	if msgID == "" {
		t.Error("message_id is empty")
	}
}

// --- Update Room ---

func TestHandleUpdateRoom_UpdatesName(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"old","members":["user1","user2"]}`)

	req := authedReq(http.MethodPatch, "/rooms/"+roomID, `{"name":"new"}`, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["name"] != "new" {
		t.Errorf("name = %q, want %q", resp["name"], "new")
	}
}

func TestHandleUpdateRoom_NotFound(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	req := authedReq(http.MethodPatch, "/rooms/bad-id", `{"name":"x"}`, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Delete Message ---

func TestHandleDeleteMessage_OwnerCanDelete(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	msgID := sendMessage(t, mux, "user1", roomID, "hello")

	req := authedReq(http.MethodDelete, "/rooms/"+roomID+"/messages/"+msgID, "", "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestHandleDeleteMessage_WrongSender(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	msgID := sendMessage(t, mux, "user1", roomID, "hello")

	req := authedReq(http.MethodDelete, "/rooms/"+roomID+"/messages/"+msgID, "", "user2")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleDeleteMessage_NotFound(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)

	req := authedReq(http.MethodDelete, "/rooms/"+roomID+"/messages/bad-id", "", "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Edit Message ---

func TestHandleEditMessage_OwnerCanEdit(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	msgID := sendMessage(t, mux, "user1", roomID, "original")

	req := authedReq(http.MethodPut, "/rooms/"+roomID+"/messages/"+msgID, `{"content":"edited"}`, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["content"] != "edited" {
		t.Errorf("content = %q, want %q", resp["content"], "edited")
	}
}

func TestHandleEditMessage_WrongSender(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	msgID := sendMessage(t, mux, "user1", roomID, "original")

	req := authedReq(http.MethodPut, "/rooms/"+roomID+"/messages/"+msgID, `{"content":"hacked"}`, "user2")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleEditMessage_EmptyContent(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	msgID := sendMessage(t, mux, "user1", roomID, "original")

	req := authedReq(http.MethodPut, "/rooms/"+roomID+"/messages/"+msgID, `{"content":""}`, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- Get Messages ---

func TestHandleGetMessages(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	sendMessage(t, mux, "user1", roomID, "msg1")
	sendMessage(t, mux, "user1", roomID, "msg2")

	req := authedReq(http.MethodGet, "/rooms/"+roomID+"/messages", "", "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	msgs := resp["messages"].([]interface{})
	if len(msgs) != 2 {
		t.Errorf("got %d messages, want 2", len(msgs))
	}
}

// --- Ack ---

func TestHandleAck(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	roomID := createRoom(t, mux, "user1", `{"type":"group","name":"g","members":["user1","user2"]}`)
	sendMessage(t, mux, "user1", roomID, "hi")

	body := `{"room_id":"` + roomID + `","seq":1}`
	req := authedReq(http.MethodPost, "/acks", body, "user2")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- Subscriptions ---

func TestHandleSubscribeAndList(t *testing.T) {
	h := newTestHandler(t)
	mux := newMux(h)

	req := authedReq(http.MethodPost, "/subscriptions", `{"topic":"orders"}`, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("subscribe status = %d; body: %s", w.Code, w.Body.String())
	}

	req = authedReq(http.MethodGet, "/subscriptions", "", "user1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	subs := resp["subscriptions"].([]interface{})
	if len(subs) != 1 {
		t.Errorf("got %d subscriptions, want 1", len(subs))
	}
}

// --- Error response shape ---

func TestErrorResponseShape(t *testing.T) {
	h := newTestHandler(t)

	// No Authorization header
	req := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	w := httptest.NewRecorder()
	h.AuthMiddleware(nopHandler)(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response missing 'error' field")
	}
	if _, ok := resp["message"]; !ok {
		t.Error("error response missing 'message' field")
	}
	if _, ok := resp["success"]; ok {
		t.Error("error response has 'success' field — old error format")
	}
}

func jsonBody(s string) *bytes.Buffer {
	return bytes.NewBufferString(s)
}

var nopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})
