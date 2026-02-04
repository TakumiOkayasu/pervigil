package config

import (
	"errors"
	"os"
)

var ErrMissingToken = errors.New("BOT_TOKEN is required")

// EnvGetter abstracts environment variable access for DI
type EnvGetter interface {
	Getenv(key string) string
}

// osEnvGetter is the default implementation using os.Getenv
type osEnvGetter struct{}

func (o *osEnvGetter) Getenv(key string) string {
	return os.Getenv(key)
}

type Config struct {
	BotToken string
	GuildID  string // optional: for faster command registration
}

// Load loads config from OS environment variables
func Load() (*Config, error) {
	return LoadWithEnv(&osEnvGetter{})
}

// LoadWithEnv loads config using the provided EnvGetter (for DI/testing)
func LoadWithEnv(env EnvGetter) (*Config, error) {
	token := env.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, ErrMissingToken
	}

	return &Config{
		BotToken: token,
		GuildID:  env.Getenv("GUILD_ID"),
	}, nil
}
