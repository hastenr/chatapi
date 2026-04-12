// Package testutil provides helpers for setting up test infrastructure.
package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/getchatapi/chatapi/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

var testDBCounter atomic.Int64

// TestJWTSecret is the shared secret used to sign JWTs in tests.
const TestJWTSecret = "test-jwt-secret"

// NewTestDB returns a fully migrated in-memory SQLite database.
// The connection is closed automatically when the test ends.
//
// Uses a named shared-cache in-memory database so that multiple connections
// within the same test all share the same schema and data, avoiding the
// per-connection isolation of plain ":memory:" SQLite databases.
func NewTestDB(t *testing.T) *db.DB {
	t.Helper()

	// Each test gets a unique DB name so they don't interfere with each other.
	id := testDBCounter.Add(1)
	dsn := fmt.Sprintf("file:testdb_%d?mode=memory&cache=shared&_busy_timeout=5000", id)

	database, err := db.New(dsn)
	if err != nil {
		t.Fatalf("testutil.NewTestDB: open: %v", err)
	}

	if err := db.RunMigrations(database); err != nil {
		database.Close()
		t.Fatalf("testutil.NewTestDB: migrations: %v", err)
	}

	t.Cleanup(func() { database.Close() })

	return database
}

// SignJWT signs a JWT with TestJWTSecret and the given subject (user ID).
// The token is valid for 5 minutes — long enough for any test.
func SignJWT(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(TestJWTSecret))
	if err != nil {
		panic("testutil.SignJWT: " + err.Error())
	}
	return signed
}
