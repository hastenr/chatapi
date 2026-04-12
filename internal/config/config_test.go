package config_test

import (
	"testing"

	"github.com/getchatapi/chatapi/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret = %q, want empty by default", cfg.JWTSecret)
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("AllowedOrigins = %v, want empty by default", cfg.AllowedOrigins)
	}
}

func TestLoad_EnvOverridesDefaults(t *testing.T) {
	t.Setenv("LISTEN_ADDR", ":9090")
	t.Setenv("JWT_SECRET", "mysecret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9090")
	}
	if cfg.JWTSecret != "mysecret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "mysecret")
	}
}

func TestLoad_AllowedOriginsFromEnv(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins len = %d, want 2", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "https://app.example.com" {
		t.Errorf("AllowedOrigins[0] = %q", cfg.AllowedOrigins[0])
	}
	if cfg.AllowedOrigins[1] != "https://admin.example.com" {
		t.Errorf("AllowedOrigins[1] = %q", cfg.AllowedOrigins[1])
	}
}

func TestValidate_MissingJWTSecret(t *testing.T) {
	cfg := &config.Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with empty JWTSecret: want error, got nil")
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := &config.Config{JWTSecret: "secret"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}
