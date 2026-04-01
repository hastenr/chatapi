package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the ChatAPI service
type Config struct {
	// Server configuration
	ListenAddr           string
	DataDir              string
	LogDir               string

	// Database configuration
	DatabaseDSN          string

	// Worker configuration
	WorkerInterval       time.Duration
	RetryMaxAttempts     int
	RetryInterval        time.Duration

	// Shutdown configuration
	ShutdownDrainTimeout time.Duration

	// Logging
	LogLevel             string

	// Auth configuration
	// JWTSecret is the HS256 secret used to validate tokens issued by the
	// deployer's backend. Set via JWT_SECRET environment variable.
	JWTSecret string

	// Webhook configuration (fired when a message arrives for an offline user)
	WebhookURL    string
	WebhookSecret string

	// WebSocket configuration
	// Comma-separated list of allowed origins. Use "*" to allow all (dev only).
	AllowedOrigins          []string
	MaxConnectionsPerUser   int
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:            getEnv("LISTEN_ADDR", ":8080"),
		DataDir:               getEnv("DATA_DIR", "/var/chatapi"),
		LogDir:                getEnv("LOG_DIR", "/var/log/chatapi"),
		DatabaseDSN:           getEnv("DATABASE_DSN", "file:chatapi.db?_journal_mode=WAL&_busy_timeout=5000"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		RetryMaxAttempts:      getEnvAsInt("RETRY_MAX_ATTEMPTS", 5),
		ShutdownDrainTimeout:  getEnvAsDuration("SHUTDOWN_DRAIN_TIMEOUT", 10*time.Second),
		WorkerInterval:        getEnvAsDuration("WORKER_INTERVAL", 30*time.Second),
		RetryInterval:         getEnvAsDuration("RETRY_INTERVAL", 30*time.Second),
		JWTSecret:             getEnv("JWT_SECRET", ""),
		WebhookURL:            getEnv("WEBHOOK_URL", ""),
		WebhookSecret:         getEnv("WEBHOOK_SECRET", ""),
		AllowedOrigins:        getEnvAsStringSlice("ALLOWED_ORIGINS"),
		MaxConnectionsPerUser: getEnvAsInt("WS_MAX_CONNECTIONS_PER_USER", 5),
	}

	return cfg, nil
}

// Validate checks that required configuration values are set.
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET must be set")
	}
	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsStringSlice gets a comma-separated environment variable as a string slice
func getEnvAsStringSlice(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	var result []string
	for _, s := range strings.Split(value, ",") {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, s)
		}
	}
	return result
}

// getEnvAsDuration gets an environment variable as time.Duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}