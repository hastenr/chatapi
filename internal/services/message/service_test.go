package message_test

import (
	"testing"

	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/repository/sqlite"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/services/message"
	"github.com/getchatapi/chatapi/internal/testutil"
)

// scenario sets up a room ready for message tests.
type scenario struct {
	roomID string
	msgSvc *message.Service
}

func newScenario(t *testing.T) *scenario {
	t.Helper()
	db := testutil.NewTestDB(t)

	chatroomSvc := chatroom.NewService(sqlite.NewRoomRepository(db.DB))
	room, err := chatroomSvc.CreateRoom(&models.CreateRoomRequest{
		Type:    "group",
		Name:    "general",
		Members: []string{"user1", "user2"},
	})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}

	return &scenario{
		roomID: room.RoomID,
		msgSvc: message.NewService(sqlite.NewMessageRepository(db.DB)),
	}
}

func (s *scenario) send(t *testing.T, senderID, content string) *models.Message {
	t.Helper()
	msg, err := s.msgSvc.SendMessage(s.roomID, senderID, &models.CreateMessageRequest{Content: content})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	return msg
}

// --- SendMessage ---

func TestSendMessage_AssignsSeq(t *testing.T) {
	s := newScenario(t)

	m1 := s.send(t, "user1", "first")
	m2 := s.send(t, "user1", "second")

	if m1.Seq != 1 {
		t.Errorf("first message seq = %d, want 1", m1.Seq)
	}
	if m2.Seq != 2 {
		t.Errorf("second message seq = %d, want 2", m2.Seq)
	}
}

func TestSendMessage_RoomNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := message.NewService(sqlite.NewMessageRepository(db.DB))

	_, err := svc.SendMessage("bad-room-id", "user1", &models.CreateMessageRequest{Content: "hi"})
	if err == nil {
		t.Error("expected error for nonexistent room, got nil")
	}
}

func TestSendMessage_StoresContent(t *testing.T) {
	s := newScenario(t)
	msg := s.send(t, "user1", "hello world")

	if msg.Content != "hello world" {
		t.Errorf("content = %q, want %q", msg.Content, "hello world")
	}
	if msg.SenderID != "user1" {
		t.Errorf("sender_id = %q, want %q", msg.SenderID, "user1")
	}
	if msg.MessageID == "" {
		t.Error("message_id is empty")
	}
}

// --- GetMessages ---

func TestGetMessages_ReturnsSentMessages(t *testing.T) {
	s := newScenario(t)
	s.send(t, "user1", "msg1")
	s.send(t, "user2", "msg2")

	msgs, err := s.msgSvc.GetMessages(s.roomID, 0, 50)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("got %d messages, want 2", len(msgs))
	}
}

func TestGetMessages_AfterSeqFilter(t *testing.T) {
	s := newScenario(t)
	s.send(t, "user1", "msg1") // seq 1
	s.send(t, "user1", "msg2") // seq 2
	s.send(t, "user1", "msg3") // seq 3

	msgs, err := s.msgSvc.GetMessages(s.roomID, 1, 50)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("got %d messages after seq 1, want 2", len(msgs))
	}
	if msgs[0].Seq != 2 {
		t.Errorf("first result seq = %d, want 2", msgs[0].Seq)
	}
}

func TestGetMessages_LimitRespected(t *testing.T) {
	s := newScenario(t)
	for i := 0; i < 5; i++ {
		s.send(t, "user1", "msg")
	}

	msgs, err := s.msgSvc.GetMessages(s.roomID, 0, 3)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("got %d messages, want 3", len(msgs))
	}
}

// --- GetMessage ---

func TestGetMessage_Found(t *testing.T) {
	s := newScenario(t)
	sent := s.send(t, "user1", "hello")

	got, err := s.msgSvc.GetMessage(sent.MessageID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if got.MessageID != sent.MessageID {
		t.Errorf("message_id = %q, want %q", got.MessageID, sent.MessageID)
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	s := newScenario(t)

	_, err := s.msgSvc.GetMessage("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent message, got nil")
	}
}

// --- DeleteMessage ---

func TestDeleteMessage_OwnerCanDelete(t *testing.T) {
	s := newScenario(t)
	msg := s.send(t, "user1", "to be deleted")

	seq, err := s.msgSvc.DeleteMessage(s.roomID, msg.MessageID, "user1")
	if err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}
	if seq != msg.Seq {
		t.Errorf("returned seq = %d, want %d", seq, msg.Seq)
	}

	// Verify it's gone
	_, err = s.msgSvc.GetMessage(msg.MessageID)
	if err == nil {
		t.Error("message still exists after deletion")
	}
}

func TestDeleteMessage_WrongSender(t *testing.T) {
	s := newScenario(t)
	msg := s.send(t, "user1", "mine")

	_, err := s.msgSvc.DeleteMessage(s.roomID, msg.MessageID, "user2")
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if err.Error() != "forbidden" {
		t.Errorf("error = %q, want %q", err.Error(), "forbidden")
	}
}

func TestDeleteMessage_NotFound(t *testing.T) {
	s := newScenario(t)

	_, err := s.msgSvc.DeleteMessage(s.roomID, "bad-id", "user1")
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	if err.Error() != "message not found" {
		t.Errorf("error = %q, want %q", err.Error(), "message not found")
	}
}

// --- UpdateMessage ---

func TestUpdateMessage_OwnerCanEdit(t *testing.T) {
	s := newScenario(t)
	msg := s.send(t, "user1", "original")

	updated, err := s.msgSvc.UpdateMessage(s.roomID, msg.MessageID, "user1", "edited")
	if err != nil {
		t.Fatalf("UpdateMessage: %v", err)
	}
	if updated.Content != "edited" {
		t.Errorf("content = %q, want %q", updated.Content, "edited")
	}
	if updated.MessageID != msg.MessageID {
		t.Error("message_id changed after edit")
	}
	if updated.Seq != msg.Seq {
		t.Error("seq changed after edit")
	}
}

func TestUpdateMessage_WrongSender(t *testing.T) {
	s := newScenario(t)
	msg := s.send(t, "user1", "original")

	_, err := s.msgSvc.UpdateMessage(s.roomID, msg.MessageID, "user2", "edited")
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	if err.Error() != "forbidden" {
		t.Errorf("error = %q, want %q", err.Error(), "forbidden")
	}
}

func TestUpdateMessage_NotFound(t *testing.T) {
	s := newScenario(t)

	_, err := s.msgSvc.UpdateMessage(s.roomID, "bad-id", "user1", "edited")
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	if err.Error() != "message not found" {
		t.Errorf("error = %q, want %q", err.Error(), "message not found")
	}
}

// --- Ack ---

func TestUpdateLastAck_StoresValue(t *testing.T) {
	s := newScenario(t)
	s.send(t, "user1", "msg")
	s.send(t, "user1", "msg")

	if err := s.msgSvc.UpdateLastAck("user2", s.roomID, 2); err != nil {
		t.Fatalf("UpdateLastAck: %v", err)
	}

	got, err := s.msgSvc.GetLastAckSeq("user2", s.roomID)
	if err != nil {
		t.Fatalf("GetLastAckSeq: %v", err)
	}
	if got != 2 {
		t.Errorf("last_ack = %d, want 2", got)
	}
}

func TestUpdateLastAck_DoesNotGoBackward(t *testing.T) {
	s := newScenario(t)

	if err := s.msgSvc.UpdateLastAck("user1", s.roomID, 5); err != nil {
		t.Fatalf("UpdateLastAck(5): %v", err)
	}
	// Attempt to move backward
	if err := s.msgSvc.UpdateLastAck("user1", s.roomID, 3); err != nil {
		t.Fatalf("UpdateLastAck(3): %v", err)
	}

	got, _ := s.msgSvc.GetLastAckSeq("user1", s.roomID)
	if got != 5 {
		t.Errorf("last_ack = %d after backward update, want 5", got)
	}
}

func TestGetLastAckSeq_NoRowReturnsZero(t *testing.T) {
	s := newScenario(t)

	got, err := s.msgSvc.GetLastAckSeq("never-acked-user", s.roomID)
	if err != nil {
		t.Fatalf("GetLastAckSeq: %v", err)
	}
	if got != 0 {
		t.Errorf("got %d, want 0 for user with no ack state", got)
	}
}

// --- MessagesSent counter ---

func TestMessagesSent_Increments(t *testing.T) {
	s := newScenario(t)

	before := s.msgSvc.MessagesSent()
	s.send(t, "user1", "a")
	s.send(t, "user1", "b")

	if got := s.msgSvc.MessagesSent(); got != before+2 {
		t.Errorf("MessagesSent = %d, want %d", got, before+2)
	}
}
