// Package utils contains small, dependency-free helpers used across layers.
package utils

import (
	"math"
	"time"
)

// StartOfDay returns midnight at the start of t in t's location.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// StartOfWeek returns Monday 00:00 of the week containing t (in t's location).
func StartOfWeek(t time.Time) time.Time {
	day := t.Weekday()
	// Go's Weekday: Sunday=0..Saturday=6. We want Monday-based weeks.
	offset := int(day) - int(time.Monday)
	if offset < 0 {
		offset += 7
	}
	monday := StartOfDay(t).AddDate(0, 0, -offset)
	return monday
}

// MinutesBetween returns the integer number of minutes between two times,
// rounded toward zero. Negative input returns 0.
func MinutesBetween(start, end time.Time) int {
	if !end.After(start) {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

// LocalCalendarDayDiff returns the signed number of calendar days between a
// and b in loc (b - a). Same calendar day = 0, next day = 1, previous = -1.
//
// math.Round absorbs the one-hour drift that DST transitions would otherwise
// introduce. Asia/Almaty has no DST today, but writing this defensively means
// the helper stays correct if the project ever supports another zone.
func LocalCalendarDayDiff(a, b time.Time, loc *time.Location) int {
	aMid := StartOfDay(a.In(loc))
	bMid := StartOfDay(b.In(loc))
	return int(math.Round(bMid.Sub(aMid).Hours() / 24))
}

// SameLocalDay is sugar over LocalCalendarDayDiff for the most common check.
func SameLocalDay(a, b time.Time, loc *time.Location) bool {
	return LocalCalendarDayDiff(a, b, loc) == 0
}
