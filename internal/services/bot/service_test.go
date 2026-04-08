package bot_test

import (
	"context"
	"testing"

	"github.com/hastenr/chatapi/internal/models"
	"github.com/hastenr/chatapi/internal/repository/sqlite"
	"github.com/hastenr/chatapi/internal/services/bot"
	"github.com/hastenr/chatapi/internal/services/chatroom"
	"github.com/hastenr/chatapi/internal/services/delivery"
	"github.com/hastenr/chatapi/internal/services/message"
	"github.com/hastenr/chatapi/internal/services/realtime"
	"github.com/hastenr/chatapi/internal/services/webhook"
	"github.com/hastenr/chatapi/internal/testutil"
)

func newBotSvc(t *testing.T) (*bot.Service, *message.Service, *chatroom.Service) {
	t.Helper()
	db := testutil.NewTestDB(t)

	roomRepo := sqlite.NewRoomRepository(db.DB)
	chatroomSvc := chatroom.NewService(roomRepo)
	messageSvc := message.NewService(sqlite.NewMessageRepository(db.DB))
	realtimeSvc := realtime.NewService(roomRepo, 5)
	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(sqlite.NewDeliveryRepository(db.DB), realtimeSvc, chatroomSvc, "", "", webhookSvc)
	botSvc := bot.NewService(sqlite.NewBotRepository(db.DB), messageSvc, realtimeSvc, chatroomSvc, deliverySvc)

	t.Cleanup(func() { realtimeSvc.Shutdown(context.Background()) })
	return botSvc, messageSvc, chatroomSvc
}

// --- CreateBot ---

func TestCreateBot_LLM(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	b, err := svc.CreateBot(&models.CreateBotRequest{
		Name:     "Helpful Bot",
		Mode:     "llm",
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	if b.BotID == "" {
		t.Error("BotID is empty")
	}
	if b.Name != "Helpful Bot" {
		t.Errorf("Name = %q, want %q", b.Name, "Helpful Bot")
	}
	if b.MaxContext != 20 {
		t.Errorf("MaxContext = %d, want 20 (default)", b.MaxContext)
	}
}

func TestCreateBot_External(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	b, err := svc.CreateBot(&models.CreateBotRequest{
		Name: "External Agent",
		Mode: "external",
	})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	if b.Mode != "external" {
		t.Errorf("Mode = %q, want external", b.Mode)
	}
}

func TestCreateBot_InvalidMode(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	_, err := svc.CreateBot(&models.CreateBotRequest{
		Name: "Bad Bot",
		Mode: "webhook",
	})
	if err == nil {
		t.Error("expected error for invalid mode, got nil")
	}
}

func TestCreateBot_LLMMissingProvider(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	_, err := svc.CreateBot(&models.CreateBotRequest{
		Name:  "No Provider",
		Mode:  "llm",
		Model: "gpt-4o",
	})
	if err == nil {
		t.Error("expected error for missing provider, got nil")
	}
}

func TestCreateBot_LLMMissingModel(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	_, err := svc.CreateBot(&models.CreateBotRequest{
		Name:     "No Model",
		Mode:     "llm",
		Provider: "openai",
	})
	if err == nil {
		t.Error("expected error for missing model, got nil")
	}
}

// --- GetBot ---

func TestGetBot_Found(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	created, _ := svc.CreateBot(&models.CreateBotRequest{
		Name:     "Finder Bot",
		Mode:     "llm",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-6",
	})

	got, err := svc.GetBot(created.BotID)
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}
	if got.BotID != created.BotID {
		t.Errorf("BotID mismatch: got %q want %q", got.BotID, created.BotID)
	}
	if got.Name != "Finder Bot" {
		t.Errorf("Name = %q, want %q", got.Name, "Finder Bot")
	}
}

func TestGetBot_NotFound(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	_, err := svc.GetBot("nonexistent-bot-id")
	if err == nil {
		t.Error("expected error for missing bot, got nil")
	}
}

// --- ListBots ---

func TestListBots_Empty(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	bots, err := svc.ListBots()
	if err != nil {
		t.Fatalf("ListBots: %v", err)
	}
	if len(bots) != 0 {
		t.Errorf("ListBots count = %d, want 0", len(bots))
	}
}

func TestListBots_Multiple(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	for i := 0; i < 3; i++ {
		svc.CreateBot(&models.CreateBotRequest{
			Name: "Bot", Mode: "external",
		})
	}

	bots, err := svc.ListBots()
	if err != nil {
		t.Fatalf("ListBots: %v", err)
	}
	if len(bots) != 3 {
		t.Errorf("ListBots count = %d, want 3", len(bots))
	}
}

// --- DeleteBot ---

func TestDeleteBot_Existing(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	b, _ := svc.CreateBot(&models.CreateBotRequest{Name: "Delete Me", Mode: "external"})

	if err := svc.DeleteBot(b.BotID); err != nil {
		t.Fatalf("DeleteBot: %v", err)
	}

	_, err := svc.GetBot(b.BotID)
	if err == nil {
		t.Error("expected bot to be gone after delete, but GetBot succeeded")
	}
}

func TestDeleteBot_NotFound(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	if err := svc.DeleteBot("ghost-id"); err == nil {
		t.Error("expected error deleting nonexistent bot, got nil")
	}
}

// --- IsBot ---

func TestIsBot_True(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	b, _ := svc.CreateBot(&models.CreateBotRequest{Name: "Am Bot", Mode: "external"})

	if !svc.IsBot(b.BotID) {
		t.Errorf("IsBot(%q) = false, want true", b.BotID)
	}
}

func TestIsBot_False(t *testing.T) {
	svc, _, _ := newBotSvc(t)

	if svc.IsBot("regular-user") {
		t.Error("IsBot(regular-user) = true, want false")
	}
}

// --- TriggerBots skips bot senders ---

func TestTriggerBots_SkipsBotSender(t *testing.T) {
	svc, messageSvc, chatroomSvc := newBotSvc(t)

	room, _ := chatroomSvc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"user1", "user2"},
	})

	b, _ := svc.CreateBot(&models.CreateBotRequest{Name: "Bot", Mode: "external"})
	chatroomSvc.AddMember(room.RoomID, b.BotID)

	// Message sent by the bot itself — should not trigger any bots
	msg, _ := messageSvc.SendMessage(room.RoomID, b.BotID, &models.CreateMessageRequest{Content: "I am a bot"})

	// TriggerBots must not panic and must return without doing anything
	svc.TriggerBots(room.RoomID, msg)
}

func TestTriggerBots_HumanMessage_NoBotInRoom(t *testing.T) {
	svc, messageSvc, chatroomSvc := newBotSvc(t)

	room, _ := chatroomSvc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "humans only", Members: []string{"alice", "bob"},
	})

	msg, _ := messageSvc.SendMessage(room.RoomID, "alice", &models.CreateMessageRequest{Content: "hello"})

	// Should not panic; no bots to trigger
	svc.TriggerBots(room.RoomID, msg)
}

func TestTriggerBots_ExternalBotNotAutoTriggered(t *testing.T) {
	svc, messageSvc, chatroomSvc := newBotSvc(t)

	room, _ := chatroomSvc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "room", Members: []string{"alice", "bob"},
	})

	extBot, _ := svc.CreateBot(&models.CreateBotRequest{Name: "Ext", Mode: "external"})
	chatroomSvc.AddMember(room.RoomID, extBot.BotID)

	msg, _ := messageSvc.SendMessage(room.RoomID, "alice", &models.CreateMessageRequest{Content: "hey"})

	// External bots are not auto-triggered — TriggerBots should silently skip
	svc.TriggerBots(room.RoomID, msg)
}
