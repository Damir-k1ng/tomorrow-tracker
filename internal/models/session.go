package models

import "time"

// Session represents a single study session for a user.
// While IsActive is true, EndedAt and DurationMinutes are zero.
type Session struct {
	ID              int64
	UserID          int64
	StartedAt       time.Time
	EndedAt         time.Time
	DurationMinutes int
	IsActive        bool
	CreatedAt       time.Time
}
