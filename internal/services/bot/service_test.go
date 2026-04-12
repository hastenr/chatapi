package bot_test

import (
	"testing"

	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/repository/sqlite"
	"github.com/getchatapi/chatapi/internal/services/bot"
	"github.com/getchatapi/chatapi/internal/testutil"
)

func newBotSvc(t *testing.T) *bot.Service {
	t.Helper()
	db := testutil.NewTestDB(t)
	return bot.NewService(sqlite.NewBotRepository(db.DB), sqlite.NewMessageRepository(db.DB), nil, "", "")
}

// testBotReq returns a valid CreateBotRequest with the given name.
func testBotReq(name string) *models.CreateBotRequest {
	return &models.CreateBotRequest{
		Name:         name,
		LLMBaseURL:   "https://api.openai.com/v1",
		LLMAPIKeyEnv: "OPENAI_API_KEY",
		Model:        "gpt-4o",
	}
}

// --- CreateBot ---

func TestCreateBot(t *testing.T) {
	svc := newBotSvc(t)

	b, err := svc.CreateBot(testBotReq("Helpful Bot"))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	if b.BotID == "" {
		t.Error("BotID is empty")
	}
	if b.Name != "Helpful Bot" {
		t.Errorf("Name = %q, want %q", b.Name, "Helpful Bot")
	}
}

func TestCreateBot_MissingName(t *testing.T) {
	svc := newBotSvc(t)

	req := testBotReq("")
	_, err := svc.CreateBot(req)
	if err == nil {
		t.Error("expected error for missing name, got nil")
	}
}

func TestCreateBot_MissingLLMBaseURL(t *testing.T) {
	svc := newBotSvc(t)

	req := testBotReq("Bot")
	req.LLMBaseURL = ""
	_, err := svc.CreateBot(req)
	if err == nil {
		t.Error("expected error for missing llm_base_url, got nil")
	}
}

func TestCreateBot_MissingAPIKeyEnv(t *testing.T) {
	svc := newBotSvc(t)

	req := testBotReq("Bot")
	req.LLMAPIKeyEnv = ""
	_, err := svc.CreateBot(req)
	if err == nil {
		t.Error("expected error for missing llm_api_key_env, got nil")
	}
}

func TestCreateBot_MissingModel(t *testing.T) {
	svc := newBotSvc(t)

	req := testBotReq("Bot")
	req.Model = ""
	_, err := svc.CreateBot(req)
	if err == nil {
		t.Error("expected error for missing model, got nil")
	}
}

// --- GetBot ---

func TestGetBot_Found(t *testing.T) {
	svc := newBotSvc(t)

	created, err := svc.CreateBot(testBotReq("Finder Bot"))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

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
	svc := newBotSvc(t)

	_, err := svc.GetBot("nonexistent-bot-id")
	if err == nil {
		t.Error("expected error for missing bot, got nil")
	}
}

// --- ListBots ---

func TestListBots_Empty(t *testing.T) {
	svc := newBotSvc(t)

	bots, err := svc.ListBots()
	if err != nil {
		t.Fatalf("ListBots: %v", err)
	}
	if len(bots) != 0 {
		t.Errorf("ListBots count = %d, want 0", len(bots))
	}
}

func TestListBots_Multiple(t *testing.T) {
	svc := newBotSvc(t)

	for i := 0; i < 3; i++ {
		svc.CreateBot(testBotReq("Bot"))
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
	svc := newBotSvc(t)

	b, err := svc.CreateBot(testBotReq("Delete Me"))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	if err := svc.DeleteBot(b.BotID); err != nil {
		t.Fatalf("DeleteBot: %v", err)
	}

	_, err = svc.GetBot(b.BotID)
	if err == nil {
		t.Error("expected bot to be gone after delete, but GetBot succeeded")
	}
}

func TestDeleteBot_NotFound(t *testing.T) {
	svc := newBotSvc(t)

	if err := svc.DeleteBot("ghost-id"); err == nil {
		t.Error("expected error deleting nonexistent bot, got nil")
	}
}

// --- IsBot ---

func TestIsBot_True(t *testing.T) {
	svc := newBotSvc(t)

	b, err := svc.CreateBot(testBotReq("Am Bot"))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	if !svc.IsBot(b.BotID) {
		t.Errorf("IsBot(%q) = false, want true", b.BotID)
	}
}

func TestIsBot_False(t *testing.T) {
	svc := newBotSvc(t)

	if svc.IsBot("regular-user") {
		t.Error("IsBot(regular-user) = true, want false")
	}
}
