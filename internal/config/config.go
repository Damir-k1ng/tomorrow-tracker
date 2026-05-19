// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// WebhookPath is the fixed URL path the Telegram webhook is served on. The
// path is NOT secret — authentication is the secret-token header — so it is
// safe for it to appear in logs and routing tables.
const WebhookPath = "/telegram/webhook"

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
	MiniAppURL         string // public HTTPS URL of the Mini App; "" disables the bot menu button
	WebhookSecret      string // Telegram webhook secret token; "" → long-polling mode
	WebhookURL         string // full public URL Telegram posts updates to (webhook mode only)
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

	// Mini App URL for the bot's chat menu button. Explicit MINI_APP_URL wins;
	// otherwise fall back to Railway's injected public domain, so production
	// works with zero extra configuration. Empty → the menu button is skipped.
	miniAppURL := getEnv("MINI_APP_URL", "")
	if miniAppURL == "" {
		if domain := os.Getenv("RAILWAY_PUBLIC_DOMAIN"); domain != "" {
			miniAppURL = "https://" + domain
		}
	}

	// Webhook mode is opt-in: setting WEBHOOK_SECRET switches the bot from
	// long-polling to a Telegram webhook served on the HTTP server. Empty →
	// long-polling, the default.
	webhookSecret := getEnv("WEBHOOK_SECRET", "")
	var webhookURL string
	if webhookSecret != "" {
		if !isValidWebhookSecret(webhookSecret) {
			return nil, errors.New("WEBHOOK_SECRET must be 1-256 chars of A-Z, a-z, 0-9, _ or -")
		}
		if miniAppURL == "" {
			return nil, errors.New("WEBHOOK_SECRET is set but no public URL is available (set MINI_APP_URL or deploy on Railway)")
		}
		webhookURL = strings.TrimRight(miniAppURL, "/") + WebhookPath
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
		Environment:   getEnv("APP_ENV", "development"),
		MiniAppURL:    miniAppURL,
		WebhookSecret: webhookSecret,
		WebhookURL:    webhookURL,
	}, nil
}

// isValidWebhookSecret reports whether s satisfies Telegram's secret_token
// rules: 1-256 characters, each one of A-Z, a-z, 0-9, '_' or '-'.
func isValidWebhookSecret(s string) bool {
	if len(s) < 1 || len(s) > 256 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
