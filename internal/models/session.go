package models

import (
	"encoding/json"
	"time"
)

// Session represents a single study session for a user.
// While IsActive is true, EndedAt and DurationMinutes are zero.
//
// IsValid is true for an honest session and false once an admin invalidates it
// (anti-cheat). An invalid finished session does NOT count toward the user's
// streak. AntiCheatFlags is a JSONB array of whitelisted evidence flags; it is
// always a valid JSON array (never null) — the empty array is "[]".
type Session struct {
	ID              int64
	UserID          int64
	StartedAt       time.Time
	EndedAt         time.Time
	DurationMinutes int
	IsActive        bool
	CreatedAt       time.Time
	IsValid         bool
	AntiCheatFlags  json.RawMessage
}
