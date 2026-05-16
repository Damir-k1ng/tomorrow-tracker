package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

// MyHours handles the "⏱ Мои часы" button.
func (h *Handlers) MyHours(ctx context.Context, msg *tgbotapi.Message) error {
	userID, err := h.ensureUser(ctx, msg.From)
	if err != nil {
		return err
	}

	progress, err := h.sessions.Progress(ctx, userID)
	if err != nil {
		return err
	}

	status := "⚪ Нет активной сессии"
	if progress.HasActiveSession {
		status = fmt.Sprintf("🟢 Сессия активна\n(с %s)",
			utils.FormatTimeHHMM(progress.ActiveStartedAt.Hour(), progress.ActiveStartedAt.Minute()))
	}

	var b strings.Builder
	fmt.Fprintf(&b,
		"📊 Твой прогресс\n\n⏱ Сегодня:\n%s\n\n🔥 За неделю:\n%s\n\n🎯 Осталось до 30 часов:\n%s\n\nСтатус:\n%s",
		utils.FormatDuration(progress.TodayMinutes),
		utils.FormatDuration(progress.WeekMinutes),
		utils.FormatDuration(progress.RemainingMinutes),
		status,
	)

	// Streak info is best-effort — a transient read failure should not block
	// the main "my hours" view.
	if cur, best, err := h.streaks.Snapshot(ctx, userID); err == nil {
		if cur > 0 || best > 0 {
			fmt.Fprintf(&b, "\n\n🔥 Стрик: %s\n🏆 Лучший стрик: %s",
				utils.FormatDays(cur), utils.FormatDays(best))
		}
	} else {
		h.log.Error("streak snapshot failed", slog.Int64("user_id", userID), slog.String("error", err.Error()))
	}

	return h.reply(msg.Chat.ID, b.String(), nil)
}
