package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	DatabaseURL   string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("can't load .env: %w", err)
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return Config{}, errors.New("TELEGRAM_TOKEN is not set")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is not set")
	}

	return Config{TelegramToken: token, DatabaseURL: databaseURL}, nil
}
