package models

import "time"

// User represents a Telegram user registered in the bot.
//
// Streak fields:
//   - CurrentStreak is the number of consecutive study days ending at LastStudyAt.
//   - BestStreak is the all-time maximum CurrentStreak ever reached.
//   - LastStudyAt is the end-time of the most recent session that counted toward
//     the streak (>= 30 minutes). Nil when the user has no qualifying session yet.
//
// All streak comparisons happen in Asia/Almaty (the configured app timezone),
// not in the stored UTC value.
type User struct {
	ID            int64
	TelegramID    int64
	Username      string
	FirstName     string
	CreatedAt     time.Time
	CurrentStreak int
	BestStreak    int
	LastStudyAt   *time.Time
}
