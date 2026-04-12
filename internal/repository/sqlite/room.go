package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/getchatapi/chatapi/internal/models"
)

// SQLiteRoomRepository implements repository.RoomRepository using SQLite.
type SQLiteRoomRepository struct {
	db *sql.DB
}

// NewRoomRepository creates a new SQLiteRoomRepository.
func NewRoomRepository(db *sql.DB) *SQLiteRoomRepository {
	return &SQLiteRoomRepository{db: db}
}

func scanRoom(row interface {
	Scan(dest ...interface{}) error
}) (*models.Room, error) {
	var room models.Room
	var uniqueKey, name, metadata sql.NullString
	err := row.Scan(
		&room.RoomID,
		&room.Type,
		&uniqueKey,
		&name,
		&room.LastSeq,
		&metadata,
		&room.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	room.UniqueKey = uniqueKey.String
	room.Name = name.String
	room.Metadata = metadata.String
	return &room, nil
}

// GetByID retrieves a room by ID.
func (r *SQLiteRoomRepository) GetByID(roomID string) (*models.Room, error) {
	row := r.db.QueryRow(
		`SELECT room_id, type, unique_key, name, last_seq, metadata, created_at FROM rooms WHERE room_id = ?`,
		roomID,
	)
	room, err := scanRoom(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("room not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return room, nil
}

// GetByUniqueKey retrieves a room by unique key. Returns nil, nil if not found.
func (r *SQLiteRoomRepository) GetByUniqueKey(uniqueKey string) (*models.Room, error) {
	row := r.db.QueryRow(
		`SELECT room_id, type, unique_key, name, last_seq, metadata, created_at FROM rooms WHERE unique_key = ?`,
		uniqueKey,
	)
	room, err := scanRoom(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get room by unique key: %w", err)
	}
	return room, nil
}

// Create inserts a new room record.
func (r *SQLiteRoomRepository) Create(room *models.Room) error {
	if room.UniqueKey != "" {
		_, err := r.db.Exec(
			`INSERT INTO rooms (room_id, type, unique_key, name, metadata, last_seq) VALUES (?, ?, ?, ?, ?, ?)`,
			room.RoomID, room.Type, room.UniqueKey, room.Name, room.Metadata, room.LastSeq,
		)
		if err != nil {
			return fmt.Errorf("failed to create room: %w", err)
		}
	} else {
		_, err := r.db.Exec(
			`INSERT INTO rooms (room_id, type, name, metadata, last_seq) VALUES (?, ?, ?, ?, ?)`,
			room.RoomID, room.Type, room.Name, room.Metadata, room.LastSeq,
		)
		if err != nil {
			return fmt.Errorf("failed to create room: %w", err)
		}
	}
	return nil
}

// Update updates a room's name and/or metadata.
func (r *SQLiteRoomRepository) Update(roomID string, req *models.UpdateRoomRequest) error {
	var setParts []string
	var args []interface{}

	if req.Name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Metadata != nil {
		setParts = append(setParts, "metadata = ?")
		args = append(args, *req.Metadata)
	}
	if len(setParts) == 0 {
		return nil
	}

	args = append(args, roomID)
	query := "UPDATE rooms SET " + strings.Join(setParts, ", ") + " WHERE room_id = ?"

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("room not found")
	}
	return nil
}

// GetUserRooms returns all rooms that a user is a member of.
func (r *SQLiteRoomRepository) GetUserRooms(userID string) ([]*models.Room, error) {
	rows, err := r.db.Query(`
		SELECT r.room_id, r.type, r.unique_key, r.name, r.last_seq, r.metadata, r.created_at
		FROM rooms r
		JOIN room_members rm ON r.room_id = rm.chatroom_id
		WHERE rm.user_id = ?
		ORDER BY r.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

// AddMember adds a single member to a room.
func (r *SQLiteRoomRepository) AddMember(roomID, userID string) error {
	_, err := r.db.Exec(
		`INSERT INTO room_members (chatroom_id, user_id, role) VALUES (?, ?, 'member')`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}
	return nil
}

// AddMembers adds multiple members to a room using a transaction with INSERT OR IGNORE.
func (r *SQLiteRoomRepository) AddMembers(roomID string, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, userID := range userIDs {
		_, err = tx.Exec(
			`INSERT OR IGNORE INTO room_members (chatroom_id, user_id, role) VALUES (?, ?, 'member')`,
			roomID, userID,
		)
		if err != nil {
			return fmt.Errorf("failed to add member %s: %w", userID, err)
		}
	}

	return tx.Commit()
}

// RemoveMember removes a member from a room.
func (r *SQLiteRoomRepository) RemoveMember(roomID, userID string) error {
	result, err := r.db.Exec(
		`DELETE FROM room_members WHERE chatroom_id = ? AND user_id = ?`,
		roomID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("member not found in room")
	}
	return nil
}

// GetMembers retrieves all members of a room.
func (r *SQLiteRoomRepository) GetMembers(roomID string) ([]*models.RoomMember, error) {
	rows, err := r.db.Query(
		`SELECT chatroom_id, user_id, role, joined_at FROM room_members WHERE chatroom_id = ? ORDER BY joined_at`,
		roomID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	defer rows.Close()

	var members []*models.RoomMember
	for rows.Next() {
		var member models.RoomMember
		if err := rows.Scan(&member.ChatroomID, &member.UserID, &member.Role, &member.JoinedAt); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, &member)
	}
	return members, nil
}

// GetMemberIDs retrieves the list of user IDs who are members of a room.
func (r *SQLiteRoomRepository) GetMemberIDs(roomID string) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT user_id FROM room_members WHERE chatroom_id = ?`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		members = append(members, userID)
	}
	return members, rows.Err()
}
