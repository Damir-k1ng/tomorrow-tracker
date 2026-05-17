package utils

import (
	"testing"
	"time"
)

var almaty = mustLoadAlmaty()

func mustLoadAlmaty() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		panic(err)
	}
	return loc
}

// day builds an end-time in Asia/Almaty for terse table setup.
func day(y, m, d, hh, mm int) time.Time {
	return time.Date(y, time.Month(m), d, hh, mm, 0, 0, almaty)
}

func TestStreakFromSessions(t *testing.T) {
	cases := []struct {
		name        string
		endedAts    []time.Time
		wantCurrent int
		wantBest    int
		wantLast    time.Time
		wantAny     bool
	}{
		{
			name:    "no qualifying sessions clears the streak",
			wantAny: false,
		},
		{
			name:        "single session",
			endedAts:    []time.Time{day(2026, 5, 16, 21, 0)},
			wantCurrent: 1, wantBest: 1,
			wantLast: day(2026, 5, 16, 21, 0), wantAny: true,
		},
		{
			name: "multiple sessions same day count once",
			endedAts: []time.Time{
				day(2026, 5, 16, 9, 0), day(2026, 5, 16, 21, 30),
			},
			wantCurrent: 1, wantBest: 1,
			wantLast: day(2026, 5, 16, 21, 30), wantAny: true,
		},
		{
			name: "three consecutive days",
			endedAts: []time.Time{
				day(2026, 5, 14, 20, 0), day(2026, 5, 15, 20, 0), day(2026, 5, 16, 20, 0),
			},
			wantCurrent: 3, wantBest: 3,
			wantLast: day(2026, 5, 16, 20, 0), wantAny: true,
		},
		{
			name: "gap leaves a longer earlier run as best",
			endedAts: []time.Time{
				day(2026, 5, 10, 20, 0), day(2026, 5, 11, 20, 0),
				day(2026, 5, 12, 20, 0), day(2026, 5, 13, 20, 0),
				day(2026, 5, 15, 20, 0), day(2026, 5, 16, 20, 0),
			},
			wantCurrent: 2, wantBest: 4,
			wantLast: day(2026, 5, 16, 20, 0), wantAny: true,
		},
		{
			name: "final day isolated after a long run",
			endedAts: []time.Time{
				day(2026, 5, 12, 20, 0), day(2026, 5, 13, 20, 0),
				day(2026, 5, 14, 20, 0), day(2026, 5, 16, 20, 0),
			},
			wantCurrent: 1, wantBest: 3,
			wantLast: day(2026, 5, 16, 20, 0), wantAny: true,
		},
		{
			name: "out-of-order input is sorted before computing",
			endedAts: []time.Time{
				day(2026, 5, 16, 20, 0), day(2026, 5, 14, 20, 0), day(2026, 5, 15, 20, 0),
			},
			wantCurrent: 3, wantBest: 3,
			wantLast: day(2026, 5, 16, 20, 0), wantAny: true,
		},
		{
			name: "session crossing midnight counts on its end-day",
			endedAts: []time.Time{
				day(2026, 5, 15, 23, 30), day(2026, 5, 16, 0, 30),
			},
			wantCurrent: 2, wantBest: 2,
			wantLast: day(2026, 5, 16, 0, 30), wantAny: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current, best, last, any := StreakFromSessions(tc.endedAts, almaty)
			if any != tc.wantAny {
				t.Fatalf("hasAny: got %v want %v", any, tc.wantAny)
			}
			if current != tc.wantCurrent || best != tc.wantBest {
				t.Errorf("got current=%d best=%d, want current=%d best=%d",
					current, best, tc.wantCurrent, tc.wantBest)
			}
			if tc.wantAny && !last.Equal(tc.wantLast) {
				t.Errorf("last: got %v want %v", last, tc.wantLast)
			}
			if !tc.wantAny && !last.IsZero() {
				t.Errorf("expected zero last-study-at, got %v", last)
			}
		})
	}
}
