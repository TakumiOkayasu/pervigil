package config

import (
	"errors"
	"testing"
)

// mapEnvGetter returns values from a map
type mapEnvGetter struct {
	values map[string]string
}

func (e *mapEnvGetter) Getenv(key string) string {
	return e.values[key]
}

func TestLoad_Success(t *testing.T) {
	env := &mapEnvGetter{
		values: map[string]string{
			"BOT_TOKEN": "test-token-123",
			"GUILD_ID":  "guild-456",
		},
	}

	cfg, err := LoadWithEnv(env)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.BotToken != "test-token-123" {
		t.Errorf("expected BotToken 'test-token-123', got '%s'", cfg.BotToken)
	}
	if cfg.GuildID != "guild-456" {
		t.Errorf("expected GuildID 'guild-456', got '%s'", cfg.GuildID)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	env := &mapEnvGetter{
		values: map[string]string{},
	}

	_, err := LoadWithEnv(env)
	if err == nil {
		t.Fatal("expected error for missing BOT_TOKEN")
	}

	if !errors.Is(err, ErrMissingToken) {
		t.Errorf("expected ErrMissingToken, got %v", err)
	}
}

func TestLoad_OptionalGuildID(t *testing.T) {
	env := &mapEnvGetter{
		values: map[string]string{
			"BOT_TOKEN": "test-token",
		},
	}

	cfg, err := LoadWithEnv(env)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.GuildID != "" {
		t.Errorf("expected empty GuildID, got '%s'", cfg.GuildID)
	}
}
