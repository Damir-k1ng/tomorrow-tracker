package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
)

// User-facing API handlers (/api/v1/user/*). Every handler here runs behind
// the user middleware chain, so an authenticated caller is always present in
// the request context. The surface is strictly read-only in this phase.
//
// Responses use typed DTO structs (not map[string]any) so the JSON contract
// is explicit and compiler-checked.

// --- typed DTOs --------------------------------------------------------------

// sessionResponse is the user-facing shape of a study session. Internal
// bookkeeping (anti-cheat flags, created_at) is deliberately omitted — a user
// sees only their own session timeline. EndedAt is null while the session runs.
type sessionResponse struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	DurationMinutes int        `json:"duration_minutes"`
	IsActive        bool       `json:"is_active"`
	IsValid         bool       `json:"is_valid"`
}

func newSessionResponse(s models.Session) sessionResponse {
	var endedAt *time.Time
	if !s.EndedAt.IsZero() {
		e := s.EndedAt
		endedAt = &e
	}
	return sessionResponse{
		ID:              s.ID,
		UserID:          s.UserID,
		StartedAt:       s.StartedAt,
		EndedAt:         endedAt,
		DurationMinutes: s.DurationMinutes,
		IsActive:        s.IsActive,
		IsValid:         s.IsValid,
	}
}

// progressResponse is the caller's study progress for the current local day
// and week. All values are server-computed in the configured timezone; the
// Mini App renders them but never derives its own. WeeklyTargetMinutes lets the
// client draw progress toward the goal without hardcoding it.
type progressResponse struct {
	TodayMinutes        int `json:"today_minutes"`
	WeekMinutes         int `json:"week_minutes"`
	RemainingMinutes    int `json:"remaining_minutes"`
	WeeklyTargetMinutes int `json:"weekly_target_minutes"`
}

// userMeResponse is the GET /api/v1/user/me payload. It is also the Mini App's
// session-recovery source: after a reopen/reconnect the client re-fetches /me
// and restores active-session state from ActiveSession — never from a local
// timer.
type userMeResponse struct {
	ID            int64            `json:"id"`
	TelegramID    int64            `json:"telegram_id"`
	FirstName     string           `json:"first_name"`
	Role          string           `json:"role"`
	CurrentStreak int              `json:"current_streak"`
	BestStreak    int              `json:"best_streak"`
	TotalMinutes  int              `json:"total_minutes"`
	TotalSessions int64            `json:"total_sessions"`
	LastStudyAt   *time.Time       `json:"last_study_at"`
	ActiveSession *sessionResponse `json:"active_session"`
	Progress      progressResponse `json:"progress"`
}

// pageMeta is the pagination block shared by paginated user listings.
type pageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// newPageMeta builds the pagination block from a parsed page request + total.
func newPageMeta(p pagination, total int64) pageMeta {
	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return pageMeta{Page: p.Page, Limit: p.Limit, Total: total, TotalPages: totalPages}
}

// sessionsResponse is the GET /api/v1/user/sessions payload.
type sessionsResponse struct {
	Items      []sessionResponse `json:"items"`
	Pagination pageMeta          `json:"pagination"`
}

// leaderboardEntryResponse is one row of the weekly Top-N.
type leaderboardEntryResponse struct {
	Rank      int    `json:"rank"`
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	Minutes   int    `json:"minutes"`
	IsCurrent bool   `json:"is_current"`
}

// leaderboardMeResponse is the caller's own standing this week. Found is false
// when the caller has no recorded minutes in the current week.
type leaderboardMeResponse struct {
	Found   bool   `json:"found"`
	Rank    int    `json:"rank"`
	Minutes int    `json:"minutes"`
	Name    string `json:"name"`
	InTop   bool   `json:"in_top"`
}

// leaderboardResponse is the GET /api/v1/user/leaderboard payload.
type leaderboardResponse struct {
	Top []leaderboardEntryResponse `json:"top"`
	Me  leaderboardMeResponse      `json:"me"`
}

// --- GET /api/v1/user/me -----------------------------------------------------

func (s *Server) handleUserMe(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil { // middleware guarantees this; defensive only
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	profile, err := s.userAPI.Profile(r.Context(), user)
	if err != nil {
		s.log.Error("api: user me failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить профиль")
		return
	}

	resp := userMeResponse{
		ID:            profile.User.ID,
		TelegramID:    profile.User.TelegramID,
		FirstName:     profile.User.FirstName,
		Role:          profile.User.Role,
		CurrentStreak: profile.User.CurrentStreak,
		BestStreak:    profile.User.BestStreak,
		TotalMinutes:  profile.TotalMinutes,
		TotalSessions: profile.TotalSessions,
		LastStudyAt:   profile.User.LastStudyAt,
	}
	if profile.ActiveSession != nil {
		active := newSessionResponse(*profile.ActiveSession)
		resp.ActiveSession = &active
	}
	resp.Progress = progressResponse{
		TodayMinutes:        profile.Progress.TodayMinutes,
		WeekMinutes:         profile.Progress.WeekMinutes,
		RemainingMinutes:    profile.Progress.RemainingMinutes,
		WeeklyTargetMinutes: profile.WeeklyTargetMinutes,
	}
	WriteSuccess(w, http.StatusOK, resp)
}

// --- GET /api/v1/user/sessions ----------------------------------------------

func (s *Server) handleUserSessions(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}
	pg, err := parsePagination(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sessions, total, err := s.userAPI.Sessions(r.Context(), user.ID, pg.Limit, pg.Offset)
	if err != nil {
		s.log.Error("api: user sessions failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить список сессий")
		return
	}

	items := make([]sessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, newSessionResponse(sess))
	}
	WriteSuccess(w, http.StatusOK, sessionsResponse{
		Items:      items,
		Pagination: newPageMeta(pg, total),
	})
}

// --- GET /api/v1/user/leaderboard -------------------------------------------

func (s *Server) handleUserLeaderboard(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	snap, err := s.userAPI.Leaderboard(r.Context(), user.ID)
	if err != nil {
		s.log.Error("api: user leaderboard failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить рейтинг")
		return
	}

	top := make([]leaderboardEntryResponse, 0, len(snap.Top))
	for _, e := range snap.Top {
		top = append(top, leaderboardEntryResponse{
			Rank:      e.Rank,
			UserID:    e.UserID,
			Name:      e.Name,
			Minutes:   e.Minutes,
			IsCurrent: e.IsCurrent,
		})
	}
	WriteSuccess(w, http.StatusOK, leaderboardResponse{
		Top: top,
		Me: leaderboardMeResponse{
			Found:   snap.User.Found,
			Rank:    snap.User.Rank,
			Minutes: snap.User.Minutes,
			Name:    snap.User.Name,
			InTop:   snap.User.InTop,
		},
	})
}

// --- session lifecycle (Phase 3C) -------------------------------------------

// streakResponse is the streak outcome attached to a finished session. It is
// null in the parent payload when the streak update itself failed — a
// deliberately non-fatal case: the session was still saved.
type streakResponse struct {
	Counted   bool `json:"counted"`
	Current   int  `json:"current"`
	Best      int  `json:"best"`
	Continued bool `json:"continued"`
	Broken    bool `json:"broken"`
	NewRecord bool `json:"new_record"`
}

// finishSessionResponse is the POST /api/v1/user/sessions/{id}/finish payload:
// the server-computed session length, the refreshed progress totals, and the
// streak evaluation.
type finishSessionResponse struct {
	SessionMinutes int              `json:"session_minutes"`
	Progress       progressResponse `json:"progress"`
	Streak         *streakResponse  `json:"streak"`
}

// --- POST /api/v1/user/sessions/start ---------------------------------------

// handleStartSession opens a new study session for the caller. The "one active
// session per user" rule is enforced both by a service pre-check and a
// database partial unique index — a duplicate start always yields 409.
func (s *Server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil { // middleware guarantees this; defensive only
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	session, err := s.userAPI.StartSession(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, services.ErrSessionAlreadyActive) {
			// Expected, benign contention (double-tap / stale UI) — Warn, not
			// Error. user_id only; no initData or secrets are ever logged.
			s.log.Warn("session: duplicate start attempt", slog.Int64("user_id", user.ID))
			WriteError(w, http.StatusConflict, "у тебя уже есть активная сессия")
			return
		}
		s.log.Error("api: start session failed",
			slog.Int64("user_id", user.ID), slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось начать сессию")
		return
	}
	s.log.Info("session: started",
		slog.Int64("user_id", user.ID), slog.Int64("session_id", session.ID))
	WriteSuccess(w, http.StatusCreated, newSessionResponse(*session))
}

// --- POST /api/v1/user/sessions/{id}/finish ---------------------------------

// handleFinishSession closes one specific session, addressed by its id. It
// enforces the full lifecycle contract: a user may only finish their own,
// still-active session. The duration is computed server-side; any client
// timer is presentation-only and never trusted.
func (s *Server) handleFinishSession(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}
	sessionID, err := parsePathID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.userAPI.FinishSession(r.Context(), user.ID, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSessionNotFound):
			WriteError(w, http.StatusNotFound, "сессия не найдена")
		case errors.Is(err, services.ErrSessionNotOwned):
			// A user attempted to finish a session they do not own — log as a
			// security-relevant event (Warn) with the ids involved.
			s.log.Warn("session: unauthorized finish attempt",
				slog.Int64("user_id", user.ID), slog.Int64("session_id", sessionID))
			WriteError(w, http.StatusForbidden, "это не твоя сессия")
		case errors.Is(err, services.ErrSessionAlreadyFinished):
			s.log.Warn("session: finish of already-finished session",
				slog.Int64("user_id", user.ID), slog.Int64("session_id", sessionID))
			WriteError(w, http.StatusConflict, "сессия уже завершена")
		default:
			s.log.Error("api: finish session failed",
				slog.Int64("user_id", user.ID), slog.Int64("session_id", sessionID),
				slog.String("error", err.Error()))
			WriteError(w, http.StatusInternalServerError, "не удалось завершить сессию")
		}
		return
	}

	// A streak-update failure is non-fatal — the session itself was saved.
	// Log it and return a null streak block rather than failing the response.
	if result.StreakErr != nil {
		s.log.Error("api: streak update failed",
			slog.Int64("user_id", user.ID),
			slog.String("error", result.StreakErr.Error()))
	}

	resp := finishSessionResponse{
		SessionMinutes: result.SessionMinutes,
		Progress: progressResponse{
			TodayMinutes:        result.Progress.TodayMinutes,
			WeekMinutes:         result.Progress.WeekMinutes,
			RemainingMinutes:    result.Progress.RemainingMinutes,
			WeeklyTargetMinutes: result.WeeklyTargetMinutes,
		},
	}
	if result.Streak != nil {
		resp.Streak = &streakResponse{
			Counted:   result.Streak.Counted,
			Current:   result.Streak.Current,
			Best:      result.Streak.Best,
			Continued: result.Streak.Continued,
			Broken:    result.Streak.Broken,
			NewRecord: result.Streak.NewRecord,
		}
	}
	s.log.Info("session: finished",
		slog.Int64("user_id", user.ID),
		slog.Int64("session_id", sessionID),
		slog.Int("duration_minutes", result.SessionMinutes))
	WriteSuccess(w, http.StatusOK, resp)
}
