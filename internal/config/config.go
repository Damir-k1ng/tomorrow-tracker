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
// Defaults are applied for everything except the bot token and database URL.
type Config struct {
	TelegramToken      string
	DatabaseURL        string
	Timezone           *time.Location
	WeeklyTargetHours  int
	LogLevel           string
	Port               string // HTTP port for the health + API endpoints
	AdminTelegramID    int64  // auto-promoted to the admin role; 0 disables
	CORSAllowedOrigins string // comma-separated allow-list; "*" permits any
	Environment        string // deployment environment: "production" enables strict checks
}

// IsProduction reports whether the bot is running in a production deployment.
// Production gates fail-fast safety checks (e.g. rejecting the placeholder SPA)
// that must not interfere with local development or tests.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Load reads .env (if present) and environment variables, returning a validated Config.
func Load() (*Config, error) {
	// .env is optional — Railway / production inject env vars directly.
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
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

	// Admin auto-promotion is opt-in: it activates only when ADMIN_TELEGRAM_ID
	// is set in the environment. The default 0 disables it — no Telegram
	// identity is hardcoded in the source tree.
	adminID, err := strconv.ParseInt(getEnv("ADMIN_TELEGRAM_ID", "0"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ADMIN_TELEGRAM_ID: %w", err)
	}

	return &Config{
		TelegramToken:     token,
		DatabaseURL:       databaseURL,
		Timezone:          loc,
		WeeklyTargetHours: target,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		// Railway injects PORT automatically; default for local runs.
		Port:               getEnv("PORT", "8080"),
		AdminTelegramID:    adminID,
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		// Railway production sets APP_ENV=production; local runs default to dev.
		Environment: getEnv("APP_ENV", "development"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
