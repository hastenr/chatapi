package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/getchatapi/chatapi/internal/models"
)

// SQLiteBotRepository implements repository.BotRepository using SQLite.
type SQLiteBotRepository struct {
	db *sql.DB
}

// NewBotRepository creates a new SQLiteBotRepository.
func NewBotRepository(db *sql.DB) *SQLiteBotRepository {
	return &SQLiteBotRepository{db: db}
}

const botColumns = `bot_id, name, llm_base_url, llm_api_key_env, model, created_at`

func scanBot(row interface {
	Scan(dest ...any) error
}) (*models.Bot, error) {
	var bot models.Bot
	err := row.Scan(
		&bot.BotID,
		&bot.Name,
		&bot.LLMBaseURL,
		&bot.LLMAPIKeyEnv,
		&bot.Model,
		&bot.CreatedAt,
	)
	return &bot, err
}

// Create registers a new bot.
func (r *SQLiteBotRepository) Create(req *models.CreateBotRequest) (*models.Bot, error) {
	bot := &models.Bot{
		BotID:        uuid.New().String(),
		Name:         req.Name,
		LLMBaseURL:   req.LLMBaseURL,
		LLMAPIKeyEnv: req.LLMAPIKeyEnv,
		Model:        req.Model,
		CreatedAt:    time.Now().UTC(),
	}

	_, err := r.db.Exec(`
		INSERT INTO bots (bot_id, name, llm_base_url, llm_api_key_env, model, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		bot.BotID, bot.Name, bot.LLMBaseURL, bot.LLMAPIKeyEnv, bot.Model, bot.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return bot, nil
}

// GetByID retrieves a bot by ID.
func (r *SQLiteBotRepository) GetByID(botID string) (*models.Bot, error) {
	row := r.db.QueryRow(`SELECT `+botColumns+` FROM bots WHERE bot_id = ?`, botID)
	bot, err := scanBot(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bot not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}
	return bot, nil
}

// List returns all registered bots.
func (r *SQLiteBotRepository) List() ([]*models.Bot, error) {
	rows, err := r.db.Query(`SELECT ` + botColumns + ` FROM bots ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list bots: %w", err)
	}
	defer rows.Close()

	var bots []*models.Bot
	for rows.Next() {
		bot, err := scanBot(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bot: %w", err)
		}
		bots = append(bots, bot)
	}
	return bots, rows.Err()
}

// Delete removes a bot by ID.
func (r *SQLiteBotRepository) Delete(botID string) error {
	result, err := r.db.Exec(`DELETE FROM bots WHERE bot_id = ?`, botID)
	if err != nil {
		return fmt.Errorf("failed to delete bot: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bot not found")
	}
	return nil
}

// Exists reports whether the given bot ID belongs to a registered bot.
func (r *SQLiteBotRepository) Exists(botID string) (bool, error) {
	var id string
	err := r.db.QueryRow(`SELECT bot_id FROM bots WHERE bot_id = ?`, botID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetBotsInRoom returns all bots that are members of the given room.
func (r *SQLiteBotRepository) GetBotsInRoom(roomID string) ([]*models.Bot, error) {
	rows, err := r.db.Query(`
		SELECT b.bot_id, b.name, b.llm_base_url, b.llm_api_key_env, b.model, b.created_at
		FROM bots b
		JOIN room_members rm ON rm.user_id = b.bot_id
		WHERE rm.chatroom_id = ?`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get bots in room: %w", err)
	}
	defer rows.Close()

	var bots []*models.Bot
	for rows.Next() {
		bot, err := scanBot(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bot: %w", err)
		}
		bots = append(bots, bot)
	}
	return bots, rows.Err()
}
