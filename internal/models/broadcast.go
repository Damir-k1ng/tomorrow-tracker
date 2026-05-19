package models

import "time"

// Broadcast lifecycle statuses.
const (
	// BroadcastRunning — the worker is (or should be) sending. A row left in
	// this state after a restart is resumed from LastProcessedUserID, never
	// abandoned.
	BroadcastRunning = "running"
	// BroadcastDone — every recipient was attempted (sent or failed).
	BroadcastDone = "done"
)

// Broadcast is one admin-initiated message fan-out to every user.
//
// LastProcessedUserID is the resume cursor: recipients are processed in
// ascending users.id order, and this records the id of the last user the
// worker attempted (whether the send succeeded or failed). A restart resumes
// strictly after it, so no user is messaged twice and none is skipped.
type Broadcast struct {
	ID                  int64
	Message             string
	Status              string
	TotalRecipients     int
	SentCount           int
	FailedCount         int
	LastProcessedUserID int64
	CreatedBy           int64 // users.id of the admin who started it
	CreatedAt           time.Time
	FinishedAt          *time.Time
}
