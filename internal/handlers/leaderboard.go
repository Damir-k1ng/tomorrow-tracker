package handlers

import (
	"context"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

const (
	leaderboardEmpty       = "🏆 Пока нет данных для рейтинга\n\nНачни первую учебную сессию 🚀"
	leaderboardDivider     = "━━━━━━━━━━"
	leaderboardNotRankedTx = "📍 Ты пока не в рейтинге\nНачни сессию, чтобы попасть в топ 🚀"
)

// Leaderboard handles the "🏆 Топ-10" button.
func (h *Handlers) Leaderboard(ctx context.Context, msg *tgbotapi.Message) error {
	userID, err := h.ensureUser(ctx, msg.From)
	if err != nil {
		return err
	}

	snap, err := h.leaderboard.Snapshot(ctx, userID)
	if err != nil {
		return err
	}

	if len(snap.Top) == 0 {
		return h.reply(msg.Chat.ID, leaderboardEmpty, nil)
	}

	return h.reply(msg.Chat.ID, renderLeaderboard(snap), nil)
}

// renderLeaderboard builds the user-facing text. Pure function — trivially
// unit-testable against fixed snapshots, no Telegram dependencies.
//
// Layout (mobile-optimized — single blank line between blocks, no padding):
//
//	🏆 Топ студентов недели
//
//	1. Damir — 10ч 42м 🔥
//	2. Alex — 8ч 11м
//	...
//	━━━━━━━━━━
//
//	📍 <position label>
//	#N — Xч YYм 🚀
func renderLeaderboard(snap *services.LeaderboardSnapshot) string {
	var b strings.Builder
	b.WriteString("🏆 Топ студентов недели\n\n")

	for _, e := range snap.Top {
		b.WriteString(formatRankLine(e))
		b.WriteString("\n")
	}

	// Always show a footer about the current user — this is the part that
	// makes the leaderboard feel personal and motivating, regardless of
	// whether they're #1 or not yet ranked.
	b.WriteString("\n")
	b.WriteString(leaderboardDivider)
	b.WriteString("\n\n")
	b.WriteString(formatUserBlock(snap.User))

	return b.String()
}

// formatUserBlock renders the personalized footer below the divider.
// Three branches:
//   - Found + InTop  → "Ты в топе:"      (celebratory framing)
//   - Found + below  → "Твоя позиция:"   (neutral framing, motivating)
//   - !Found         → friendly nudge    (no fake "#0 — 0ч 00м" line)
func formatUserBlock(u services.UserPosition) string {
	if !u.Found {
		return leaderboardNotRankedTx
	}

	label := "Твоя позиция:"
	if u.InTop {
		label = "Ты в топе:"
	}

	return "📍 " + label + "\n#" + strconv.Itoa(u.Rank) +
		" — " + utils.FormatDuration(u.Minutes) + " 🚀"
}

// formatRankLine renders one Top-N entry. A fire emoji decorates rank #1 only;
// the user's own line gets no extra decoration here — that lives in the footer.
func formatRankLine(e services.LeaderboardEntry) string {
	suffix := ""
	if e.Rank == 1 {
		suffix = " 🔥"
	}
	return strconv.Itoa(e.Rank) + ". " + e.Name + " — " + utils.FormatDuration(e.Minutes) + suffix
}
