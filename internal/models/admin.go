package models

import (
	"encoding/json"
	"time"
)

// AdminStats is the operational snapshot returned by GET /api/v1/admin/stats.
// Operational counters only — no advanced analytics.
type AdminStats struct {
	TotalUsers        int64   `json:"total_users"`
	ActiveSessions    int64   `json:"active_sessions"`
	TotalStudyHours   float64 `json:"total_study_hours"`
	CompletedSessions int64   `json:"completed_sessions"`
	BestStreak        int     `json:"best_streak"`
	AverageStreak     float64 `json:"average_streak"`
	NewUsers7d        int64   `json:"new_users_7d"`
}

// AuditLog is one immutable, append-only record from the admin_actions table.
// BeforeData / AfterData carry raw JSON so the original snapshots are preserved
// verbatim.
type AuditLog struct {
	ID           int64
	AdminID      int64
	Action       string
	EntityType   *string
	EntityID     *int64
	TargetUserID *int64
	BeforeData   json.RawMessage
	AfterData    json.RawMessage
	Reason       *string
	CreatedAt    time.Time
}

// SessionCorrection is the result of an admin session correction; it feeds
// both the API response and the structured action log. It carries the before
// and after of every correctable field plus the outcome of the deterministic,
// per-user streak recomputation that a correction may trigger.
type SessionCorrection struct {
	SessionID   int64
	OwnerID     int64
	OldDuration int
	NewDuration int
	OldIsValid  bool
	NewIsValid  bool

	// StreakRecomputed is true when the correction changed whether the session
	// qualifies for the streak, forcing a recomputation of the owner's streak.
	// When false, CurrentStreak/BestStreak hold the owner's unchanged values.
	StreakRecomputed bool
	CurrentStreak    int
	BestStreak       int
}

// UserDetails is the composed payload for GET /api/v1/admin/users/:id.
// RecentSessions is intentionally bounded (last 20) — never the full history.
type UserDetails struct {
	User           *User
	TotalMinutes   int
	ActiveSession  *Session
	RecentSessions []Session
}
