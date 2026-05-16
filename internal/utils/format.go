package utils

import (
	"fmt"
	"strconv"
)

// FormatDuration renders a number of minutes as "Xч YYм" — the short Russian
// style used throughout the bot UI. Minutes are zero-padded to two digits so
// values line up nicely in the leaderboard ("10ч 03м").
// Negative values are clamped to zero.
func FormatDuration(minutes int) string {
	if minutes < 0 {
		minutes = 0
	}
	h := minutes / 60
	m := minutes % 60
	return fmt.Sprintf("%dч %02dм", h, m)
}

// FormatTimeHHMM formats a time as 24-hour HH:MM. Caller is responsible for
// passing the time in the desired location.
func FormatTimeHHMM(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// FormatDays renders a day count with the correct Russian noun form.
//
// Examples: 1 → "1 день", 2 → "2 дня", 5 → "5 дней", 21 → "21 день".
//
// Standard Russian rule:
//   - n%100 in 11..14 → "дней" (special teens)
//   - n%10 == 1       → "день"
//   - n%10 in 2..4    → "дня"
//   - else            → "дней"
func FormatDays(n int) string {
	return strconv.Itoa(n) + " " + dayNoun(n)
}

func dayNoun(n int) string {
	if n < 0 {
		n = -n
	}
	mod100 := n % 100
	if mod100 >= 11 && mod100 <= 14 {
		return "дней"
	}
	switch n % 10 {
	case 1:
		return "день"
	case 2, 3, 4:
		return "дня"
	default:
		return "дней"
	}
}
