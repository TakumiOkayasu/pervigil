package config

import (
	"errors"
	"os"
)

type Config struct {
	BotToken string
	GuildID  string // optional: for faster command registration
}

func Load() (*Config, error) {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, errors.New("BOT_TOKEN is required")
	}

	return &Config{
		BotToken: token,
		GuildID:  os.Getenv("GUILD_ID"),
	}, nil
}
