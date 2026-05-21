// Package handlers contains the message handlers invoked by the bot router.
// Handlers translate Telegram updates into service calls and render replies.
package handlers

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
)

// Handlers bundles every dependency the message handlers need so the bot
// router can dispatch with a single struct reference.
type Handlers struct {
	bot         *tgbotapi.BotAPI
	users       *services.UserService
	sessions    *services.SessionService
	leaderboard *services.LeaderboardService
	streaks     *services.StreakService
	ai          *services.AIService // nil when PIONEER_API_KEY is unset
	log         *slog.Logger
}

// New builds the handler bundle. ai may be nil — the /ask handler degrades
// gracefully to an "AI disabled" message instead of crashing.
func New(
	bot *tgbotapi.BotAPI,
	users *services.UserService,
	sessions *services.SessionService,
	leaderboard *services.LeaderboardService,
	streaks *services.StreakService,
	ai *services.AIService,
	log *slog.Logger,
) *Handlers {
	return &Handlers{
		bot:         bot,
		users:       users,
		sessions:    sessions,
		leaderboard: leaderboard,
		streaks:     streaks,
		ai:          ai,
		log:         log,
	}
}

// reply sends a Markdown-free message back to the chat that produced msg.
// Replies are intentionally plain text — emoji do all the visual work, and
// avoiding Markdown removes a whole class of escaping bugs.
func (h *Handlers) reply(chatID int64, text string, kb *tgbotapi.ReplyKeyboardMarkup) error {
	out := tgbotapi.NewMessage(chatID, text)
	if kb != nil {
		out.ReplyMarkup = *kb
	}
	if _, err := h.bot.Send(out); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// ensureUser is called on every interaction so the local users table always
// reflects the latest Telegram username/first_name.
func (h *Handlers) ensureUser(ctx context.Context, from *tgbotapi.User) (int64, error) {
	u, err := h.users.EnsureUser(ctx, from.ID, from.UserName, from.FirstName)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}
