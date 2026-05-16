package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

// StartSession handles the "▶️ Начать сессию" button.
func (h *Handlers) StartSession(ctx context.Context, msg *tgbotapi.Message) error {
	userID, err := h.ensureUser(ctx, msg.From)
	if err != nil {
		return err
	}

	session, err := h.sessions.Start(ctx, userID)
	if err != nil {
		if errors.Is(err, services.ErrSessionAlreadyActive) {
			return h.reply(msg.Chat.ID,
				"⚠️ У тебя уже есть активная сессия.\n\nЗаверши её, чтобы начать новую.",
				nil)
		}
		return err
	}

	startedLocal := session.StartedAt.In(h.sessions.Location())
	text := fmt.Sprintf(
		"✅ Учебная сессия началась\n\n⏰ Время начала:\n%s\n\nУдачного обучения 🚀",
		utils.FormatTimeHHMM(startedLocal.Hour(), startedLocal.Minute()),
	)
	return h.reply(msg.Chat.ID, text, nil)
}

// EndSession handles the "⏹ Завершить сессию" button.
//
// Orchestration: close the session, then evaluate the streak. The streak
// service is the authority on streak logic — handlers never compute it.
// A streak update failure is logged but does not block the success reply,
// so transient DB issues never make a successful session look broken.
func (h *Handlers) EndSession(ctx context.Context, msg *tgbotapi.Message) error {
	userID, err := h.ensureUser(ctx, msg.From)
	if err != nil {
		return err
	}

	result, err := h.sessions.Finish(ctx, userID)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSession) {
			return h.reply(msg.Chat.ID,
				"⚪ У тебя нет активной сессии.\n\nНажми ▶️ Начать сессию, чтобы запустить таймер.",
				nil)
		}
		return err
	}

	streakSection := ""
	streakUpd, streakErr := h.streaks.RecordCompletedSession(ctx, userID, result.SessionMinutes, result.EndedAt)
	if streakErr != nil {
		// Don't fail the user-facing reply for a streak persistence error —
		// the session itself was saved successfully.
		h.log.Error("streak update failed", slog.Int64("user_id", userID), slog.String("error", streakErr.Error()))
	} else {
		streakSection = formatStreakSection(streakUpd)
	}

	text := fmt.Sprintf(
		"✅ Сессия завершена\n\n⏱ Сессия:\n%s\n\n📅 Сегодня:\n%s\n\n🔥 За неделю:\n%s\n\n🎯 Осталось до цели:\n%s",
		utils.FormatDuration(result.SessionMinutes),
		utils.FormatDuration(result.Progress.TodayMinutes),
		utils.FormatDuration(result.Progress.WeekMinutes),
		utils.FormatDuration(result.Progress.RemainingMinutes),
	)
	if streakSection != "" {
		text += "\n\n" + streakSection
	}
	return h.reply(msg.Chat.ID, text, nil)
}

// formatStreakSection renders the streak block appended to the session-end
// message. Returns "" when there is nothing meaningful to show (e.g. a
// sub-30-minute session that did not affect the streak).
//
// Display priority:
//   1. New all-time record (>1 day)            → 🏆 celebration
//   2. Streak just broke and restarted at 1    → ⚠️ + reassurance
//   3. Streak continued (or first day at 1)    → 🔥 standard line
//   4. Same-day re-completion                  → 🔥 standard line (no change)
func formatStreakSection(u *services.StreakUpdate) string {
	if u == nil || !u.Counted {
		return ""
	}

	switch {
	case u.NewRecord && u.Current > 1:
		return fmt.Sprintf("🏆 Новый рекорд:\n%s подряд 🚀", utils.FormatDays(u.Current))
	case u.Broken:
		return fmt.Sprintf("⚠️ Серия прервалась\n\n🔥 Начинаем новую серию: %s", utils.FormatDays(u.Current))
	default:
		return fmt.Sprintf("🔥 Серия: %s подряд", utils.FormatDays(u.Current))
	}
}
