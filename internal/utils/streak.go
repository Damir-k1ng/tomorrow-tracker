package utils

import (
	"sort"
	"time"
)

// StreakFromSessions deterministically rebuilds a user's materialized streak
// from the end-times of all their qualifying study sessions (finished, valid,
// and at least the minimum length — the caller filters that upstream).
//
// It is the recomputation counterpart of the incremental streak logic: an
// admin correction that changes whether a session qualifies cannot be applied
// incrementally, so the whole streak is rebuilt from scratch for that one user.
//
// Rules, identical to the incremental path:
//   - A calendar day "counts" if at least one qualifying session ENDED that day.
//   - Days are bucketed in loc, so a session crossing midnight is attributed to
//     the day it ended in.
//   - current is the length of the consecutive run ending at the most recent
//     counted day; best is the longest consecutive run ever.
//   - lastStudyAt is the end-time of the most recent qualifying session.
//
// With no qualifying sessions it returns (0, 0, zero, false) — the caller then
// clears the streak (current=0, best=0, last_study_at=NULL).
func StreakFromSessions(endedAts []time.Time, loc *time.Location) (current, best int, lastStudyAt time.Time, hasAny bool) {
	if len(endedAts) == 0 {
		return 0, 0, time.Time{}, false
	}

	// Collect distinct local calendar days and track the latest end-time.
	seen := make(map[int64]struct{}, len(endedAts))
	days := make([]time.Time, 0, len(endedAts))
	for _, t := range endedAts {
		day := StartOfDay(t.In(loc))
		key := day.Unix()
		if _, dup := seen[key]; !dup {
			seen[key] = struct{}{}
			days = append(days, day)
		}
		if t.After(lastStudyAt) {
			lastStudyAt = t
		}
	}

	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })

	// Walk the sorted days: a gap of exactly one day extends the run, anything
	// else restarts it. current ends as the run length of the final block.
	current, best = 1, 1
	for i := 1; i < len(days); i++ {
		if LocalCalendarDayDiff(days[i-1], days[i], loc) == 1 {
			current++
		} else {
			current = 1
		}
		if current > best {
			best = current
		}
	}
	return current, best, lastStudyAt, true
}
