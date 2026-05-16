package services

import (
	"context"
	"fmt"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

// LeaderboardSize is how many entries the Top-N shows.
const LeaderboardSize = 10

// LeaderboardEntry is one row rendered in the Top-N list.
type LeaderboardEntry struct {
	Rank      int
	UserID    int64
	Name      string // first_name, falling back to @username, then "Student"
	Minutes   int
	IsCurrent bool // true when this entry is the requesting user
}

// UserPosition describes the requesting user's standing this week.
// InTop is true when the user already appears in the Top-N list.
type UserPosition struct {
	Rank    int
	Minutes int
	Name    string
	Found   bool // false when the user has zero recorded minutes this week
	InTop   bool
}

// LeaderboardSnapshot is everything the handler needs to render one screen.
type LeaderboardSnapshot struct {
	Top  []LeaderboardEntry
	User UserPosition
}

// LeaderboardService computes the weekly Top-N and the requesting user's rank.
type LeaderboardService struct {
	repo     repositories.SessionRepository
	location *time.Location
}

// NewLeaderboardService wires a LeaderboardService.
func NewLeaderboardService(repo repositories.SessionRepository, loc *time.Location) *LeaderboardService {
	return &LeaderboardService{repo: repo, location: loc}
}

// Snapshot returns the current week's leaderboard plus the requesting user's
// rank. Window: [Monday 00:00 local .. now], with each session capped at 12h.
func (s *LeaderboardService) Snapshot(ctx context.Context, currentUserID int64) (*LeaderboardSnapshot, error) {
	now := time.Now().In(s.location)
	weekStart := utils.StartOfWeek(now)

	totals, err := s.repo.WeeklyTotals(ctx, weekStart, now)
	if err != nil {
		return nil, fmt.Errorf("weekly totals: %w", err)
	}

	snap := &LeaderboardSnapshot{}

	// totals is already sorted by the repo (minutes DESC, created_at ASC),
	// so a single pass produces both the Top-N slice and the user's rank.
	for i, t := range totals {
		rank := i + 1
		isCurrent := t.UserID == currentUserID
		name := pickName(t.FirstName, t.Username)

		if rank <= LeaderboardSize {
			snap.Top = append(snap.Top, LeaderboardEntry{
				Rank:      rank,
				UserID:    t.UserID,
				Name:      name,
				Minutes:   t.Minutes,
				IsCurrent: isCurrent,
			})
		}

		if isCurrent {
			snap.User = UserPosition{
				Rank:    rank,
				Minutes: t.Minutes,
				Name:    name,
				Found:   true,
				InTop:   rank <= LeaderboardSize,
			}
		}
	}

	return snap, nil
}

// pickName implements the spec fallback: first_name → username → "Student".
// Original capitalization is preserved — names are never uppercased or otherwise
// transformed, so "Damir" stays "Damir" and "alex" stays "alex".
func pickName(firstName, username string) string {
	if firstName != "" {
		return firstName
	}
	if username != "" {
		return username
	}
	return "Student"
}
