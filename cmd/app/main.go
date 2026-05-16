// Tomorrow Tracker — Telegram bot that helps Tomorrow School Astana students
// track their weekly study hours during the Pool phase.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/damirkabdulla/tomorrow-tracker/internal/bot"
	"github.com/damirkabdulla/tomorrow-tracker/internal/config"
	"github.com/damirkabdulla/tomorrow-tracker/internal/database"
	"github.com/damirkabdulla/tomorrow-tracker/internal/handlers"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
	"github.com/damirkabdulla/tomorrow-tracker/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// We can't use the configured logger before the level is known, so
		// this single bootstrap error goes to plain stderr.
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).
			Error("config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info("tomorrow-tracker starting",
		slog.String("timezone", cfg.Timezone.String()),
		slog.Int("weekly_target_hours", cfg.WeeklyTargetHours),
	)

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Error("database init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("database close failed", slog.String("error", err.Error()))
		}
	}()

	api, err := bot.Connect(cfg.TelegramToken, log)
	if err != nil {
		log.Error("telegram connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	userRepo := repositories.NewUserRepository(db)
	sessionRepo := repositories.NewSessionRepository(db)

	userSvc := services.NewUserService(userRepo)
	sessionSvc := services.NewSessionService(sessionRepo, cfg.Timezone, cfg.WeeklyTargetHours)
	leaderboardSvc := services.NewLeaderboardService(sessionRepo, cfg.Timezone)
	streakSvc := services.NewStreakService(userRepo, cfg.Timezone)

	h := handlers.New(api, userSvc, sessionSvc, leaderboardSvc, streakSvc, log)
	router := bot.NewRouter(h)
	tgBot := bot.New(api, router, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tgBot.Run(ctx)
	log.Info("tomorrow-tracker stopped cleanly")
}
