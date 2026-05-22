package handlers

import (
	"context"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/menu"
)

// telegramMessageLimit is the maximum single Telegram message length in
// runes. Long AI replies are chunked at this boundary so nothing is silently
// dropped by the API.
const telegramMessageLimit = 4096

const aiModeWelcome = `🤖 <b>AI-ментор включён</b>

Теперь просто пиши свой вопрос или вставляй код — я отвечу.
Можно по-русски или по-английски.

Например:
• <i>почему мой цикл for бесконечный?</i>
• <i>что делает оператор &amp; в Go?</i>
• <i>дай скелет HTTP-сервера</i>

Кнопки внизу: 🗑 очистить диалог или 🚪 выйти.`

const aiDisabled = "🤖 AI-ментор временно недоступен. Попробуй позже или используй кнопки меню."

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
		return h.reply(msg.Chat.ID, aiDisabled, nil)
	}

	question := strings.TrimSpace(msg.CommandArguments())
	if question == "" {
		return h.reply(msg.Chat.ID,
			"🤖 AI-ментор\n\n"+
				"Напиши вопрос или вставь код после команды:\n"+
				"/ask почему мой цикл for бесконечный?\n\n"+
				"Или нажми кнопку \"🤖 AI-ментор\" в меню — там удобнее, не нужно писать /ask перед каждым вопросом.\n\n"+
				"/clear — очистить историю диалога",
			nil)
	}

	return h.askAI(ctx, msg, question)
}

// AskFree handles a free-form message from a user who is already in AI chat
// mode (entered via the "🤖 AI-ментор" button). The whole message text is
// the question — no command prefix to strip.
func (h *Handlers) AskFree(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	if h.ai == nil {
		return h.reply(msg.Chat.ID, aiDisabled, nil)
	}
	question := strings.TrimSpace(msg.Text)
	if question == "" {
		return nil
	}
	return h.askAI(ctx, msg, question)
}

// askAI performs the actual round-trip to the model, renders the reply as
// Telegram HTML, and chunks the response if it overflows the per-message
// limit. Shared by both the /ask command and the free-form chat mode.
func (h *Handlers) askAI(ctx context.Context, msg *tgbotapi.Message, question string) error {
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

	html := markdownToTelegramHTML(answer)
	for _, chunk := range splitForTelegram(html, telegramMessageLimit) {
		if err := h.replyHTML(msg.Chat.ID, chunk); err != nil {
			// Telegram rejects malformed HTML with 400. Fall back to plain
			// text so the student still receives the answer.
			h.log.Warn("html parse rejected, sending plain text",
				slog.String("error", err.Error()))
			if err2 := h.reply(msg.Chat.ID, answer, nil); err2 != nil {
				return err2
			}
			return nil
		}
	}
	return nil
}

// EnterAIMode is called when the user taps "🤖 AI-ментор". It flips the
// per-user flag in AIService and swaps the keyboard to the AI-mode buttons.
func (h *Handlers) EnterAIMode(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	if h.ai == nil {
		return h.reply(msg.Chat.ID, aiDisabled, nil)
	}
	h.ai.EnterAIMode(msg.From.ID)
	kb := menu.AIMode()
	return h.replyHTMLWithKB(msg.Chat.ID, aiModeWelcome, &kb)
}

// ExitAIMode is called when the user taps "🚪 Выйти из AI". The free-form
// flag drops and the main keyboard returns. History is intentionally
// preserved — re-entering AI mode resumes the previous conversation.
func (h *Handlers) ExitAIMode(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	if h.ai != nil {
		h.ai.ExitAIMode(msg.From.ID)
	}
	kb := menu.Main()
	return h.reply(msg.Chat.ID, "👋 Вышли из AI-ментора. История диалога сохранена — нажми кнопку \"🤖 AI-ментор\" чтобы продолжить.", &kb)
}

// ClearAI handles the /clear command and the "🗑 Очистить диалог" button.
// Drops the user's AI dialog history but keeps them in AI mode.
func (h *Handlers) ClearAI(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	if h.ai == nil {
		return h.reply(msg.Chat.ID, "🤖 AI-ментор не настроен — очищать нечего.", nil)
	}
	h.ai.Clear(msg.From.ID)
	// Keep the AI-mode keyboard if the user is in AI mode; otherwise no kb.
	if h.ai.IsInAIMode(msg.From.ID) {
		kb := menu.AIMode()
		return h.reply(msg.Chat.ID, "🗑 История AI-диалога очищена.", &kb)
	}
	return h.reply(msg.Chat.ID, "🗑 История AI-диалога очищена.", nil)
}

// splitForTelegram chops s into pieces no longer than limit runes. It
// prefers to split on a newline boundary near the end of each chunk so
// HTML tags (used by askAI for code formatting) don't get cut in half.
// Falling back to a hard rune cut keeps the function safe on inputs that
// have no newlines.
func splitForTelegram(s string, limit int) []string {
	if limit <= 0 || s == "" {
		return []string{s}
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return []string{s}
	}
	parts := make([]string, 0, (len(runes)+limit-1)/limit)
	for i := 0; i < len(runes); {
		end := i + limit
		if end >= len(runes) {
			parts = append(parts, string(runes[i:]))
			break
		}
		// Walk back to the nearest newline within the last 25% of the chunk.
		// This avoids cutting in the middle of an HTML tag for typical AI
		// output, which is heavy on newlines between paragraphs and blocks.
		minSplit := i + (limit * 3 / 4)
		split := end
		for j := end; j > minSplit; j-- {
			if runes[j-1] == '\n' {
				split = j
				break
			}
		}
		parts = append(parts, string(runes[i:split]))
		i = split
	}
	return parts
}
