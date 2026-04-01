// Package auth provides JWT validation for ChatAPI.
//
// ChatAPI validates JWTs signed by the deployer's backend using a shared
// secret (HS256). The deployer issues tokens to their authenticated users;
// ChatAPI trusts them without managing its own user records.
//
// Required claim: "sub" (string) — used as the ChatAPI user ID.
// Optional claims: standard exp/iat/iss — validated if present.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

// UserIDKey is the context key under which the authenticated user ID is stored.
const UserIDKey contextKey = "user_id"

// ValidateJWT parses and validates a JWT signed with the given HS256 secret.
// Returns the subject (user ID) on success.
func ValidateJWT(secret, tokenStr string) (userID string, err error) {
	if secret == "" {
		return "", errors.New("JWT_SECRET is not configured")
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", errors.New("token missing sub claim")
	}

	return sub, nil
}

// WithUserID returns a copy of ctx with the user ID stored under UserIDKey.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// UserIDFromContext retrieves the user ID stored by WithUserID.
func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(UserIDKey).(string)
	return uid, ok && uid != ""
}
