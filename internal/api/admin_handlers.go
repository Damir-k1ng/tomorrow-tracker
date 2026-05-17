package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// auditSortColumns whitelists sortable columns for the audit-log listing.
var auditSortColumns = map[string]string{"created_at": "created_at"}

// --- GET /api/v1/admin/stats -------------------------------------------------

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.admin.Stats(r.Context())
	if err != nil {
		s.log.Error("api: admin stats failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить статистику")
		return
	}
	WriteSuccess(w, http.StatusOK, stats)
}

// --- GET /api/v1/admin/users -------------------------------------------------

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePagination(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	col, desc := parseSort(r, userSortColumns, "created_at", true)

	users, total, err := s.admin.ListUsers(r.Context(), repositories.UserListParams{
		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
		SortColumn: col,
		SortDesc:   desc,
		Limit:      pg.Limit,
		Offset:     pg.Offset,
	})
	if err != nil {
		s.log.Error("api: list users failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить список пользователей")
		return
	}

	items := make([]map[string]any, 0, len(users))
	for _, u := range users {
		items = append(items, userDTO(u))
	}
	WriteSuccess(w, http.StatusOK, pageData(items, pg, total))
}

// --- GET /api/v1/admin/users/{id} --------------------------------------------

func (s *Server) handleAdminUserDetails(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	details, err := s.admin.UserDetails(r.Context(), id)
	if errors.Is(err, repositories.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	if err != nil {
		s.log.Error("api: user details failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить данные пользователя")
		return
	}
	WriteSuccess(w, http.StatusOK, userDetailsDTO(details))
}

// --- PATCH /api/v1/admin/sessions/{id} ---------------------------------------

func (s *Server) handleAdminPatchSession(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	patch, err := parseSessionPatch(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	admin := userFromContext(r.Context())
	if admin == nil { // middleware guarantees this; defensive only
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	corr, err := s.admin.CorrectSession(r.Context(), admin.ID, id, repositories.SessionPatch{
		DurationMinutes: patch.DurationMinutes,
		IsValid:         patch.IsValid,
		AntiCheatFlags:  patch.AntiCheatFlags,
		SetFlags:        patch.SetFlags,
		Reason:          patch.Reason,
	})
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		WriteError(w, http.StatusNotFound, "сессия не найдена")
		return
	case errors.Is(err, repositories.ErrSessionActive):
		WriteError(w, http.StatusConflict, "активную сессию нельзя редактировать")
		return
	case err != nil:
		s.log.Error("api: correct session failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось скорректировать сессию")
		return
	}

	s.logAdminAction(admin.ID, "PATCH_SESSION", "session", corr.SessionID)
	WriteSuccess(w, http.StatusOK, map[string]any{
		"session_id":           corr.SessionID,
		"user_id":              corr.OwnerID,
		"old_duration_minutes": corr.OldDuration,
		"new_duration_minutes": corr.NewDuration,
		"old_is_valid":         corr.OldIsValid,
		"new_is_valid":         corr.NewIsValid,
		"streak_recomputed":    corr.StreakRecomputed,
		"current_streak":       corr.CurrentStreak,
		"best_streak":          corr.BestStreak,
	})
}

// --- GET /api/v1/admin/audit-logs --------------------------------------------

func (s *Server) handleAdminListAuditLogs(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePagination(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	_, desc := parseSort(r, auditSortColumns, "created_at", true)

	logs, total, err := s.admin.ListAuditLogs(r.Context(), repositories.AuditListParams{
		Action:   optionalString(r, "action"),
		AdminID:  optionalInt64(r, "admin_id"),
		SortDesc: desc,
		Limit:    pg.Limit,
		Offset:   pg.Offset,
	})
	if err != nil {
		s.log.Error("api: list audit logs failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось получить журнал аудита")
		return
	}

	items := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		items = append(items, auditDTO(l))
	}
	WriteSuccess(w, http.StatusOK, pageData(items, pg, total))
}

// --- GET /api/v1/admin/export/users ------------------------------------------

func (s *Server) handleAdminExportUsers(w http.ResponseWriter, r *http.Request) {
	rng, err := parseExportRange(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	admin := userFromContext(r.Context())
	if admin == nil {
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	// Audit is recorded BEFORE streaming, so an export is always logged even
	// if the client disconnects mid-download.
	if err := s.admin.RecordExport(r.Context(), admin.ID, "EXPORT_USERS", rng.From, rng.To); err != nil {
		s.log.Error("api: record export audit failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось записать аудит экспорта")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="users.csv"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "telegram_id", "username", "first_name", "role", "current_streak", "best_streak", "created_at"})
	streamErr := s.admin.StreamUsers(r.Context(), rng.From, rng.To, func(u models.User) error {
		return cw.Write([]string{
			strconv.FormatInt(u.ID, 10),
			strconv.FormatInt(u.TelegramID, 10),
			u.Username,
			u.FirstName,
			u.Role,
			strconv.Itoa(u.CurrentStreak),
			strconv.Itoa(u.BestStreak),
			u.CreatedAt.Format(time.RFC3339),
		})
	})
	cw.Flush()
	if err := errors.Join(streamErr, cw.Error()); err != nil {
		// The 200 + partial body is already sent; only logging remains.
		s.log.Error("api: export users stream failed", slog.String("error", err.Error()))
		return
	}
	s.logAdminAction(admin.ID, "EXPORT_USERS", "export", 0)
}

// --- GET /api/v1/admin/export/sessions ---------------------------------------

func (s *Server) handleAdminExportSessions(w http.ResponseWriter, r *http.Request) {
	rng, err := parseExportRange(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	admin := userFromContext(r.Context())
	if admin == nil {
		WriteError(w, http.StatusUnauthorized, "требуется авторизация")
		return
	}

	if err := s.admin.RecordExport(r.Context(), admin.ID, "EXPORT_SESSIONS", rng.From, rng.To); err != nil {
		s.log.Error("api: record export audit failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось записать аудит экспорта")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="sessions.csv"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "user_id", "started_at", "ended_at", "duration_minutes", "is_active", "is_valid", "anti_cheat_flags", "created_at"})
	streamErr := s.admin.StreamSessions(r.Context(), rng.From, rng.To, func(sess models.Session) error {
		endedAt := ""
		if !sess.EndedAt.IsZero() {
			endedAt = sess.EndedAt.Format(time.RFC3339)
		}
		return cw.Write([]string{
			strconv.FormatInt(sess.ID, 10),
			strconv.FormatInt(sess.UserID, 10),
			sess.StartedAt.Format(time.RFC3339),
			endedAt,
			strconv.Itoa(sess.DurationMinutes),
			strconv.FormatBool(sess.IsActive),
			strconv.FormatBool(sess.IsValid),
			string(sess.AntiCheatFlags),
			sess.CreatedAt.Format(time.RFC3339),
		})
	})
	cw.Flush()
	if err := errors.Join(streamErr, cw.Error()); err != nil {
		s.log.Error("api: export sessions stream failed", slog.String("error", err.Error()))
		return
	}
	s.logAdminAction(admin.ID, "EXPORT_SESSIONS", "export", 0)
}

// --- helpers -----------------------------------------------------------------

// logAdminAction emits the structured admin-action log line (Step 12).
func (s *Server) logAdminAction(adminID int64, action, entity string, entityID int64) {
	attrs := []any{
		slog.Int64("admin_id", adminID),
		slog.String("action", action),
		slog.String("entity", entity),
	}
	if entityID > 0 {
		attrs = append(attrs, slog.Int64("entity_id", entityID))
	}
	s.log.Info("ADMIN_ACTION", attrs...)
}

// pageData wraps a list payload with pagination metadata.
func pageData(items any, p pagination, total int64) map[string]any {
	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return map[string]any{
		"items": items,
		"pagination": map[string]any{
			"page":        p.Page,
			"limit":       p.Limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}
}

func userDTO(u models.User) map[string]any {
	return map[string]any{
		"id":             u.ID,
		"telegram_id":    u.TelegramID,
		"username":       u.Username,
		"first_name":     u.FirstName,
		"role":           u.Role,
		"current_streak": u.CurrentStreak,
		"best_streak":    u.BestStreak,
		"last_study_at":  u.LastStudyAt,
		"created_at":     u.CreatedAt,
	}
}

func sessionDTO(sess models.Session) map[string]any {
	var endedAt any
	if !sess.EndedAt.IsZero() {
		endedAt = sess.EndedAt
	}
	return map[string]any{
		"id":               sess.ID,
		"user_id":          sess.UserID,
		"started_at":       sess.StartedAt,
		"ended_at":         endedAt,
		"duration_minutes": sess.DurationMinutes,
		"is_active":        sess.IsActive,
		"is_valid":         sess.IsValid,
		"anti_cheat_flags": rawOrNil(sess.AntiCheatFlags),
		"created_at":       sess.CreatedAt,
	}
}

func auditDTO(a models.AuditLog) map[string]any {
	return map[string]any{
		"id":             a.ID,
		"admin_id":       a.AdminID,
		"action":         a.Action,
		"entity_type":    a.EntityType,
		"entity_id":      a.EntityID,
		"target_user_id": a.TargetUserID,
		"before_data":    rawOrNil(a.BeforeData),
		"after_data":     rawOrNil(a.AfterData),
		"reason":         a.Reason,
		"created_at":     a.CreatedAt,
	}
}

func userDetailsDTO(d *models.UserDetails) map[string]any {
	recent := make([]map[string]any, 0, len(d.RecentSessions))
	for _, sess := range d.RecentSessions {
		recent = append(recent, sessionDTO(sess))
	}
	var active any
	if d.ActiveSession != nil {
		active = sessionDTO(*d.ActiveSession)
	}
	return map[string]any{
		"profile": userDTO(*d.User),
		"streak": map[string]any{
			"current":       d.User.CurrentStreak,
			"best":          d.User.BestStreak,
			"last_study_at": d.User.LastStudyAt,
		},
		"total_minutes":   d.TotalMinutes,
		"total_hours":     math.Round(float64(d.TotalMinutes)/6) / 10,
		"active_session":  active,
		"recent_sessions": recent,
	}
}

// rawOrNil renders a JSONB column as embedded JSON, or null when absent.
func rawOrNil(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
