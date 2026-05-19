// Tomorrow Tracker — Telegram bot that helps Tomorrow School Astana students
// track their weekly study hours during the Pool phase.
//
// Startup order: load env → connect PostgreSQL → ping → migrate → start bot.
// The bot polls Telegram; a tiny HTTP health endpoint keeps Railway happy.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	assets "github.com/damirkabdulla/tomorrow-tracker"
	"github.com/damirkabdulla/tomorrow-tracker/internal/api"
	"github.com/damirkabdulla/tomorrow-tracker/internal/bot"
	"github.com/damirkabdulla/tomorrow-tracker/internal/config"
	"github.com/damirkabdulla/tomorrow-tracker/internal/database"
	"github.com/damirkabdulla/tomorrow-tracker/internal/handlers"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
	"github.com/damirkabdulla/tomorrow-tracker/internal/web"
	"github.com/damirkabdulla/tomorrow-tracker/pkg/logger"
)

func main() {
	// 1. Load environment / config.
	cfg, err := config.Load()
	if err != nil {
		// The configured logger needs a level from config, so this single
		// bootstrap error goes to plain stderr.
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).
			Error("config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info("tomorrow-tracker starting",
		slog.String("timezone", cfg.Timezone.String()),
		slog.Int("weekly_target_hours", cfg.WeeklyTargetHours),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2-4. Connect PostgreSQL, ping, run migrations (all inside database.New).
	pool, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		// Fatal: the bot cannot run without its database. Loud, clear, exit.
		log.Error("FATAL: database initialization failed",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database connected and migrated")

	// Connect to Telegram.
	tgAPI, err := bot.Connect(cfg.TelegramToken, log)
	if err != nil {
		log.Error("FATAL: telegram connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Wire repositories → services → handlers → router → bot.
	userRepo := repositories.NewUserRepository(pool)
	sessionRepo := repositories.NewSessionRepository(pool)
	adminRepo := repositories.NewAdminRepository(pool, cfg.Timezone)

	userSvc := services.NewUserService(userRepo, cfg.AdminTelegramID)
	sessionSvc := services.NewSessionService(sessionRepo, cfg.Timezone, cfg.WeeklyTargetHours)
	leaderboardSvc := services.NewLeaderboardService(sessionRepo, cfg.Timezone)
	streakSvc := services.NewStreakService(userRepo, cfg.Timezone)
	adminSvc := services.NewAdminService(adminRepo, sessionRepo, userRepo)
	userAPISvc := services.NewUserAPIService(sessionRepo, leaderboardSvc, sessionSvc, streakSvc)

	// Broadcast: admin message fan-out. Throttled to broadcastsPerSecond, well
	// under Telegram's ~30/s global limit, so a running broadcast leaves
	// headroom for the live bot's own replies.
	const broadcastsPerSecond = 15
	broadcastRepo := repositories.NewBroadcastRepository(pool)
	broadcastSvc := services.NewBroadcastService(
		ctx, broadcastRepo, bot.NewSender(tgAPI, log), time.Second/broadcastsPerSecond, log)
	// Resume a broadcast left mid-flight by a previous deploy/restart.
	broadcastSvc.Resume()

	h := handlers.New(tgAPI, userSvc, sessionSvc, leaderboardSvc, streakSvc, log)
	router := bot.NewRouter(h)
	tgBot := bot.New(tgAPI, router, log, cfg.MiniAppURL)

	// Anti-stub guard: in production the embedded SPA must be a real Vite
	// build, not the committed placeholder that exists only so `go build`
	// works. Fail fast so a missing/skipped frontend build never ships.
	if err := web.EnsureRealBuild(assets.SPA, cfg.IsProduction()); err != nil {
		log.Error("FATAL: embedded frontend check failed",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Embedded Telegram Mini App SPA. A build failure here is non-fatal: the
	// API must keep serving even if the frontend bundle is missing.
	spaHandler, err := web.Handler(assets.SPA)
	if err != nil {
		log.Error("embedded SPA unavailable — serving API only",
			slog.String("error", err.Error()))
		spaHandler = nil // api.New substitutes a safe stub
	}

	// Update delivery mode is chosen by config: WEBHOOK_SECRET set → the bot
	// receives updates via a Telegram webhook served on the HTTP server;
	// unset → classic long-polling. The webhook handler must exist before the
	// server is built so it can be mounted on the mux.
	var webhookHandler http.Handler
	if cfg.WebhookSecret != "" {
		webhookHandler = tgBot.WebhookHandler(ctx, cfg.WebhookSecret)
	}

	// HTTP server: Railway health endpoint, the /api/v1 surface, the embedded
	// Mini App, and (in webhook mode) the Telegram webhook. Started before the
	// bot so the platform — and Telegram — see a healthy service immediately.
	apiSrv := api.New(cfg.Port, cfg.TelegramToken, cfg.CORSAllowedOrigins, userSvc, userAPISvc, adminSvc, broadcastSvc, spaHandler, config.WebhookPath, webhookHandler, log)
	apiSrv.Start()
	defer apiSrv.Shutdown()

	// 5. Start the bot. Both Run and RunWebhook block until ctx is cancelled.
	if cfg.WebhookSecret != "" {
		if err := tgBot.RunWebhook(ctx, cfg.WebhookURL, cfg.WebhookSecret); err != nil {
			log.Error("FATAL: webhook registration failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	} else {
		tgBot.Run(ctx)
	}
	log.Info("tomorrow-tracker stopped cleanly")
}
