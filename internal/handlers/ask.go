package handlers

import (
	"context"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// telegramMessageLimit is the maximum single Telegram message length in
// runes. Long AI replies are chunked at this boundary so nothing is silently
// dropped by the API.
const telegramMessageLimit = 4096

// Ask handles the /ask command. Usage:
//
//	/ask <question or code>
//
// The leading "/ask " is stripped; everything after is sent to the AI mentor.
// When the AI service is not configured (empty PIONEER_API_KEY at startup)
// the handler shows a graceful message instead of an error.
func (h *Handlers) Ask(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}

	if h.ai == nil {
		return h.reply(msg.Chat.ID,
			"🤖 AI-ментор временно недоступен. Попробуй позже или используй кнопки меню.",
			nil)
	}

	question := strings.TrimSpace(msg.CommandArguments())
	if question == "" {
		return h.reply(msg.Chat.ID,
			"🤖 AI-ментор\n\n"+
				"Напиши вопрос или вставь код после команды:\n"+
				"/ask почему мой цикл for бесконечный?\n\n"+
				"Команды:\n"+
				"/clear — очистить историю диалога",
			nil)
	}

	// Show "typing…" so the student knows the request is in flight — Pioneer
	// inference for an 8B model takes several seconds.
	_, _ = h.bot.Request(tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatTyping))

	answer, err := h.ai.Ask(ctx, msg.From.ID, question)
	if err != nil {
		h.log.Error("ai ask failed",
			slog.Int64("user_id", msg.From.ID),
			slog.String("error", err.Error()),
		)
		return h.reply(msg.Chat.ID,
			"❌ Не удалось получить ответ от AI. Попробуй ещё раз через минуту.",
			nil)
	}

	for _, chunk := range splitForTelegram(answer, telegramMessageLimit) {
		if err := h.reply(msg.Chat.ID, chunk, nil); err != nil {
			return err
		}
	}
	return nil
}

// ClearAI handles the /clear command — drops the user's AI dialog history.
func (h *Handlers) ClearAI(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	if h.ai == nil {
		return h.reply(msg.Chat.ID, "🤖 AI-ментор не настроен — очищать нечего.", nil)
	}
	h.ai.Clear(msg.From.ID)
	return h.reply(msg.Chat.ID, "🗑️ История AI-диалога очищена.", nil)
}

// splitForTelegram chops s into pieces no longer than limit runes. Splitting
// by runes (not bytes) keeps multi-byte characters like Cyrillic intact.
func splitForTelegram(s string, limit int) []string {
	if limit <= 0 || s == "" {
		return []string{s}
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return []string{s}
	}
	parts := make([]string, 0, (len(runes)+limit-1)/limit)
	for i := 0; i < len(runes); i += limit {
		end := i + limit
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[i:end]))
	}
	return parts
}
