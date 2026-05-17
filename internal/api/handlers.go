package api

import (
	"log/slog"
	"net/http"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// handleHealth is the Railway health endpoint. It stays plain text (not the
// JSON envelope) so any health-check probe is satisfied with a 200.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleAPINotFound is the catch-all for any unmatched /api/ path. It keeps API
// 404s as JSON so the SPA fallback never serves an HTML shell for an API route.
func handleAPINotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, "эндпоинт не найден")
}

// handleSPAUnavailable stands in for the embedded SPA when it failed to
// initialise (e.g. a missing build). It keeps the server routable instead of
// crashing — the API surface stays fully functional.
func handleSPAUnavailable(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusServiceUnavailable, "интерфейс недоступен")
}

// handleNotFound is the default for the /api/v1/user and /api/v1/admin route
// groups. Phase 1 ships the routing + middleware foundation; concrete business
// endpoints arrive in later phases. Requests still pass through CORS, rate
// limiting and auth, so the foundation is fully exercised.
func handleNotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, "эндпоинт не реализован")
}

// handleAuthVerify is the Mini App login endpoint. It verifies the Telegram
// initData, registers/refreshes the user, and returns their profile + role.
// This is real Phase 1 logic — the Mini App calls it to obtain a session.
func (s *Server) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	raw, ok := extractInitData(r)
	if !ok {
		WriteError(w, http.StatusBadRequest, "отсутствуют данные Telegram")
		return
	}

	data, err := VerifyInitData(raw, s.botToken, initDataMaxAge)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "недействительные данные авторизации")
		return
	}

	user, err := s.users.EnsureUser(r.Context(), data.User.ID, data.User.Username, data.User.FirstName)
	if err != nil {
		s.log.Error("api: ensure user failed", slog.String("error", err.Error()))
		WriteError(w, http.StatusInternalServerError, "не удалось определить пользователя")
		return
	}

	WriteSuccess(w, http.StatusOK, userPayload(user))
}

// userPayload is the JSON shape returned for an authenticated user. It is kept
// deliberately small — only what a Mini App needs to render and gate UI.
func userPayload(u *models.User) map[string]any {
	return map[string]any{
		"id":          u.ID,
		"telegram_id": u.TelegramID,
		"first_name":  u.FirstName,
		"username":    u.Username,
		"role":        u.Role,
	}
}
