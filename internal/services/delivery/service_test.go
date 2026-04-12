package delivery_test

import (
	"context"
	"testing"

	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/repository/sqlite"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/services/delivery"
	"github.com/getchatapi/chatapi/internal/services/message"
	"github.com/getchatapi/chatapi/internal/services/realtime"
	"github.com/getchatapi/chatapi/internal/services/webhook"
	"github.com/getchatapi/chatapi/internal/testutil"
)

type deliveryScenario struct {
	roomID      string
	deliverySvc *delivery.Service
	messageSvc  *message.Service
	realtimeSvc *realtime.Service
}

func newDeliveryScenario(t *testing.T) *deliveryScenario {
	t.Helper()
	db := testutil.NewTestDB(t)

	roomRepo := sqlite.NewRoomRepository(db.DB)
	chatroomSvc := chatroom.NewService(roomRepo)
	room, err := chatroomSvc.CreateRoom(&models.CreateRoomRequest{
		Type:    "group",
		Name:    "general",
		Members: []string{"user1", "user2", "user3"},
	})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}

	realtimeSvc := realtime.NewService(roomRepo, 5)
	t.Cleanup(func() { realtimeSvc.Shutdown(context.Background()) })

	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(sqlite.NewDeliveryRepository(db.DB), realtimeSvc, chatroomSvc, "", "", webhookSvc)
	messageSvc := message.NewService(sqlite.NewMessageRepository(db.DB))

	return &deliveryScenario{
		roomID:      room.RoomID,
		deliverySvc: deliverySvc,
		messageSvc:  messageSvc,
		realtimeSvc: realtimeSvc,
	}
}

// --- HandleNewMessage ---

func TestHandleNewMessage_QueuesForOfflineUsers(t *testing.T) {
	s := newDeliveryScenario(t)

	msg, err := s.messageSvc.SendMessage(s.roomID, "user1", &models.CreateMessageRequest{
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	s.deliverySvc.HandleNewMessage(s.roomID, msg)

	undelivered, err := s.messageSvc.GetUndeliveredMessages("user2", 50)
	if err != nil {
		t.Fatalf("GetUndeliveredMessages(user2): %v", err)
	}
	if len(undelivered) != 1 {
		t.Errorf("user2 undelivered count = %d, want 1", len(undelivered))
	}

	undelivered, err = s.messageSvc.GetUndeliveredMessages("user3", 50)
	if err != nil {
		t.Fatalf("GetUndeliveredMessages(user3): %v", err)
	}
	if len(undelivered) != 1 {
		t.Errorf("user3 undelivered count = %d, want 1", len(undelivered))
	}
}

func TestHandleNewMessage_SenderNotQueued(t *testing.T) {
	s := newDeliveryScenario(t)

	msg, err := s.messageSvc.SendMessage(s.roomID, "user1", &models.CreateMessageRequest{
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	s.deliverySvc.HandleNewMessage(s.roomID, msg)

	undelivered, err := s.messageSvc.GetUndeliveredMessages("user1", 50)
	if err != nil {
		t.Fatalf("GetUndeliveredMessages(user1): %v", err)
	}
	if len(undelivered) != 0 {
		t.Errorf("sender has %d undelivered entries, want 0", len(undelivered))
	}
}

// --- ProcessUndeliveredMessages ---

func TestProcessUndeliveredMessages_IncreasesAttempts(t *testing.T) {
	s := newDeliveryScenario(t)

	msg, _ := s.messageSvc.SendMessage(s.roomID, "user1", &models.CreateMessageRequest{Content: "hi"})
	s.deliverySvc.HandleNewMessage(s.roomID, msg)

	if err := s.deliverySvc.ProcessUndeliveredMessages(50); err != nil {
		t.Fatalf("ProcessUndeliveredMessages: %v", err)
	}

	undelivered, _ := s.messageSvc.GetUndeliveredMessages("user2", 50)
	if len(undelivered) == 0 {
		t.Fatal("expected undelivered entry after processing")
	}
	if undelivered[0].Attempts != 1 {
		t.Errorf("attempts = %d, want 1", undelivered[0].Attempts)
	}
}

func TestProcessUndeliveredMessages_DeliveredCounters(t *testing.T) {
	s := newDeliveryScenario(t)

	msg, _ := s.messageSvc.SendMessage(s.roomID, "user1", &models.CreateMessageRequest{Content: "hi"})
	s.deliverySvc.HandleNewMessage(s.roomID, msg)

	before := s.deliverySvc.DeliveryAttempts()
	s.deliverySvc.ProcessUndeliveredMessages(50)
	after := s.deliverySvc.DeliveryAttempts()

	if after-before != 2 {
		t.Errorf("delivery attempts delta = %d, want 2", after-before)
	}
}

// --- DeliveryAttempts / DeliveryFailures counters ---

func TestDeliveryCounters_InitiallyZero(t *testing.T) {
	s := newDeliveryScenario(t)

	if got := s.deliverySvc.DeliveryAttempts(); got != 0 {
		t.Errorf("DeliveryAttempts = %d, want 0", got)
	}
	if got := s.deliverySvc.DeliveryFailures(); got != 0 {
		t.Errorf("DeliveryFailures = %d, want 0", got)
	}
}

func TestDeliveryFailures_IncrementsWhenOffline(t *testing.T) {
	s := newDeliveryScenario(t)

	msg, _ := s.messageSvc.SendMessage(s.roomID, "user1", &models.CreateMessageRequest{Content: "hi"})
	s.deliverySvc.HandleNewMessage(s.roomID, msg)
	s.deliverySvc.ProcessUndeliveredMessages(50)

	if got := s.deliverySvc.DeliveryFailures(); got == 0 {
		t.Error("DeliveryFailures = 0 after processing offline users, want > 0")
	}
}
