package models

import "time"

// WeeklyTotal is one row of an aggregated weekly leaderboard query.
// Minutes is already capped per session and clipped to the requested window.
type WeeklyTotal struct {
	UserID    int64
	FirstName string
	Username  string
	Minutes   int
	CreatedAt time.Time // user creation time, used as deterministic tie-breaker
}
