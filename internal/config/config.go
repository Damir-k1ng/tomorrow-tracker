// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every runtime parameter the bot needs.
// Defaults are applied for everything except the bot token.
type Config struct {
	TelegramToken     string
	DatabasePath      string
	Timezone          *time.Location
	WeeklyTargetHours int
	LogLevel          string
}

// Load reads .env (if present) and environment variables, returning a validated Config.
func Load() (*Config, error) {
	// .env is optional — Docker / production typically inject env vars directly.
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	tzName := getEnv("TIMEZONE", "Asia/Almaty")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("invalid TIMEZONE %q: %w", tzName, err)
	}

	target, err := strconv.Atoi(getEnv("WEEKLY_TARGET_HOURS", "30"))
	if err != nil || target <= 0 {
		return nil, fmt.Errorf("invalid WEEKLY_TARGET_HOURS: %v", err)
	}

	return &Config{
		TelegramToken:     token,
		DatabasePath:      getEnv("DATABASE_PATH", "./storage/tomorrow.db"),
		Timezone:          loc,
		WeeklyTargetHours: target,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
