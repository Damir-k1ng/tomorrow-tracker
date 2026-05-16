package handlers

import (
	"strings"
	"testing"

	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
)

func TestRenderLeaderboard(t *testing.T) {
	top := []services.LeaderboardEntry{
		{Rank: 1, UserID: 1, Name: "Damir", Minutes: 642},  // 10ч 42м
		{Rank: 2, UserID: 2, Name: "Alex", Minutes: 491},   // 8ч 11м
		{Rank: 3, UserID: 3, Name: "Diana", Minutes: 418},  // 6ч 58м
	}

	tests := []struct {
		name     string
		snap     *services.LeaderboardSnapshot
		mustHave []string
		mustMiss []string
	}{
		{
			name: "user in top",
			snap: &services.LeaderboardSnapshot{
				Top:  top,
				User: services.UserPosition{Rank: 1, Minutes: 642, Name: "Damir", Found: true, InTop: true},
			},
			mustHave: []string{
				"🏆 Топ студентов недели",
				"1. Damir — 10ч 42м 🔥",
				"2. Alex — 8ч 11м",
				"━━━━━━━━━━",
				"📍 Ты в топе:",
				"#1 — 10ч 42м 🚀",
			},
			mustMiss: []string{"Твоя позиция:", "не в рейтинге"},
		},
		{
			name: "user below top",
			snap: &services.LeaderboardSnapshot{
				Top:  top,
				User: services.UserPosition{Rank: 17, Minutes: 252, Name: "Self", Found: true, InTop: false},
			},
			mustHave: []string{
				"📍 Твоя позиция:",
				"#17 — 4ч 12м 🚀",
			},
			mustMiss: []string{"Ты в топе:", "не в рейтинге"},
		},
		{
			name: "user not ranked",
			snap: &services.LeaderboardSnapshot{
				Top:  top,
				User: services.UserPosition{Found: false},
			},
			mustHave: []string{
				"1. Damir — 10ч 42м 🔥",
				"━━━━━━━━━━",
				"📍 Ты пока не в рейтинге",
				"Начни сессию",
			},
			mustMiss: []string{"#0", "Ты в топе:", "Твоя позиция:"},
		},
		{
			name: "fire emoji only on rank 1",
			snap: &services.LeaderboardSnapshot{
				Top:  top,
				User: services.UserPosition{Found: false},
			},
			mustMiss: []string{
				"2. Alex — 8ч 11м 🔥",
				"3. Diana — 6ч 58м 🔥",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderLeaderboard(tc.snap)
			for _, want := range tc.mustHave {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\n--- output ---\n%s", want, got)
				}
			}
			for _, banned := range tc.mustMiss {
				if strings.Contains(got, banned) {
					t.Errorf("output unexpectedly contains %q\n--- output ---\n%s", banned, got)
				}
			}
		})
	}
}
