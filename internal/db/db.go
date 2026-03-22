package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	// SQLite WAL mode supports concurrent readers; allow multiple connections
	// so background workers don't starve health checks or request handlers.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// RunMigrations runs all database migrations
func RunMigrations(db *DB) error {
	slog.Info("Running database migrations")

	// Enable WAL mode for better concurrency
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Set WAL autocheckpoint
	if _, err := db.Exec(`PRAGMA wal_autocheckpoint = 1000`); err != nil {
		return fmt.Errorf("failed to set WAL autocheckpoint: %w", err)
	}

	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Run migrations in order
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if err := runMigration(db, filename); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", filename, err)
		}
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

// createMigrationsTable creates the schema_migrations table
func createMigrationsTable(db *DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

// runMigration runs a single migration if it hasn't been applied yet
func runMigration(db *DB, filename string) error {
	// Check if migration has already been applied
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", filename).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		slog.Debug("Migration already applied", "file", filename)
		return nil
	}

	// Read migration file
	content, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	slog.Info("Applying migration", "file", filename)
	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration as applied
	if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", filename); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// CheckpointWAL performs a WAL checkpoint to merge WAL file into main database
func CheckpointWAL(db *DB) error {
	_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}