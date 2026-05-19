package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// Centralized request-validation layer. Every admin handler parses query
// params and bodies through these helpers so validation rules live in exactly
// one place. All messages are Russian (Russian-first product).

const (
	defaultPageLimit  = 20
	maxPageLimit      = 100
	maxExportDays     = 366
	maxReasonLen      = 500
	maxAntiCheatFlags = 16
	exportDateLayout  = "2006-01-02"
)

// allowedAntiCheatFlags is the fixed taxonomy of anti-cheat evidence flags an
// admin may attach to a session. Anything outside this set is rejected with a
// 400 so analytics, exports, and the audit log keep a stable, deterministic
// vocabulary instead of accumulating free-form drift.
var allowedAntiCheatFlags = map[string]struct{}{
	"manual_review":       {},
	"suspicious_duration": {},
	"rapid_restarts":      {},
	"overlap_detected":    {},
	"admin_invalidated":   {},
}

// pagination is the parsed, validated page/limit/offset for a list request.
type pagination struct {
	Page   int
	Limit  int
	Offset int
}

// parsePagination reads ?page and ?limit, applying defaults and bounds.
func parsePagination(r *http.Request) (pagination, error) {
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return pagination{}, errors.New("параметр page должен быть положительным числом")
		}
		page = n
	}

	limit := defaultPageLimit
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return pagination{}, errors.New("параметр limit должен быть положительным числом")
		}
		if n > maxPageLimit {
			n = maxPageLimit
		}
		limit = n
	}

	return pagination{Page: page, Limit: limit, Offset: (page - 1) * limit}, nil
}

// parseSort resolves ?sort against a whitelist. A "-" prefix means descending.
// allowed maps user-facing keys to SAFE SQL column literals; unknown keys fall
// back to defKey. The returned column is always a trusted literal.
func parseSort(r *http.Request, allowed map[string]string, defKey string, defDesc bool) (column string, desc bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("sort"))
	if raw == "" {
		return allowed[defKey], defDesc
	}
	desc = false
	if strings.HasPrefix(raw, "-") {
		desc = true
		raw = raw[1:]
	}
	if col, ok := allowed[raw]; ok {
		return col, desc
	}
	return allowed[defKey], defDesc
}

// userSortColumns whitelists the sortable columns for the users listing.
var userSortColumns = map[string]string{
	"created_at":     "created_at",
	"current_streak": "current_streak",
	"best_streak":    "best_streak",
}

// sessionPatch is the validated body of PATCH /api/v1/admin/sessions/:id.
// DurationMinutes and IsValid are nil when the field was absent from the
// request — a partial patch is allowed. AntiCheatFlags is set only when
// SetFlags is true, and holds a canonical (deduplicated, sorted, whitelisted)
// JSON array of flag strings. Reason is always mandatory.
type sessionPatch struct {
	DurationMinutes *int
	IsValid         *bool
	AntiCheatFlags  []byte
	SetFlags        bool
	Reason          string
}

// parseSessionPatch decodes and validates the correction body. Unknown fields
// are rejected, which structurally blocks telegram_id / ownership / timestamp
// tampering — only duration_minutes, is_valid, anti_cheat_flags and reason are
// accepted. At least one editable field must be present.
func parseSessionPatch(r *http.Request) (sessionPatch, error) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var body struct {
		DurationMinutes *int      `json:"duration_minutes"`
		IsValid         *bool     `json:"is_valid"`
		AntiCheatFlags  *[]string `json:"anti_cheat_flags"`
		Reason          string    `json:"reason"`
	}
	if err := dec.Decode(&body); err != nil {
		return sessionPatch{}, errors.New("некорректное тело запроса: разрешены только duration_minutes, is_valid, anti_cheat_flags и reason")
	}

	var out sessionPatch

	if body.DurationMinutes != nil {
		if *body.DurationMinutes < 0 || *body.DurationMinutes > repositories.MaxSessionMinutes {
			return sessionPatch{}, errors.New("duration_minutes должно быть от 0 до 720")
		}
		out.DurationMinutes = body.DurationMinutes
	}
	out.IsValid = body.IsValid

	if body.AntiCheatFlags != nil {
		flags, err := canonicalAntiCheatFlags(*body.AntiCheatFlags)
		if err != nil {
			return sessionPatch{}, err
		}
		out.AntiCheatFlags = flags
		out.SetFlags = true
	}

	if out.DurationMinutes == nil && out.IsValid == nil && !out.SetFlags {
		return sessionPatch{}, errors.New("нужно указать хотя бы одно изменяемое поле: duration_minutes, is_valid или anti_cheat_flags")
	}

	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		return sessionPatch{}, errors.New("поле reason обязательно")
	}
	if len(reason) > maxReasonLen {
		return sessionPatch{}, errors.New("reason слишком длинный (макс. 500 символов)")
	}
	out.Reason = reason
	return out, nil
}

// canonicalAntiCheatFlags validates every flag against the whitelist, removes
// duplicates, sorts the result, and marshals it to a JSON array. An unknown
// flag returns an error (→ 400). The empty input array yields "[]" — the
// canonical "no flags" value, never JSON null.
func canonicalAntiCheatFlags(raw []string) ([]byte, error) {
	if len(raw) > maxAntiCheatFlags {
		return nil, fmt.Errorf("слишком много флагов anti_cheat_flags (макс. %d)", maxAntiCheatFlags)
	}
	seen := make(map[string]struct{}, len(raw))
	flags := make([]string, 0, len(raw))
	for _, f := range raw {
		f = strings.TrimSpace(f)
		if _, ok := allowedAntiCheatFlags[f]; !ok {
			return nil, fmt.Errorf("недопустимый флаг anti_cheat_flags: %q", f)
		}
		if _, dup := seen[f]; dup {
			continue
		}
		seen[f] = struct{}{}
		flags = append(flags, f)
	}
	sort.Strings(flags)
	b, err := json.Marshal(flags)
	if err != nil {
		return nil, errors.New("не удалось обработать anti_cheat_flags")
	}
	return b, nil
}

// dateRange is the validated [From, To) export window. To is exclusive — it is
// the day AFTER the requested end date, so the end date is fully included.
type dateRange struct {
	From time.Time
	To   time.Time
}

// parseExportRange reads the mandatory ?from and ?to dates (YYYY-MM-DD) and
// enforces the maximum allowed export span.
func parseExportRange(r *http.Request) (dateRange, error) {
	q := r.URL.Query()
	fromStr, toStr := q.Get("from"), q.Get("to")
	if fromStr == "" || toStr == "" {
		return dateRange{}, errors.New("параметры from и to обязательны (формат YYYY-MM-DD)")
	}

	from, err := time.Parse(exportDateLayout, fromStr)
	if err != nil {
		return dateRange{}, errors.New("неверный формат from (ожидается YYYY-MM-DD)")
	}
	to, err := time.Parse(exportDateLayout, toStr)
	if err != nil {
		return dateRange{}, errors.New("неверный формат to (ожидается YYYY-MM-DD)")
	}
	if to.Before(from) {
		return dateRange{}, errors.New("to не может быть раньше from")
	}
	if to.Sub(from) > maxExportDays*24*time.Hour {
		return dateRange{}, errors.New("слишком большой диапазон экспорта (макс. 366 дней)")
	}

	// Make the end date inclusive by extending the upper bound to next midnight.
	return dateRange{From: from, To: to.AddDate(0, 0, 1)}, nil
}

// parsePathID extracts and validates a positive integer {id} path segment.
func parsePathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("некорректный идентификатор в пути запроса")
	}
	return id, nil
}

// optionalString returns a pointer to the query value, or nil when absent.
func optionalString(r *http.Request, key string) *string {
	if v := strings.TrimSpace(r.URL.Query().Get(key)); v != "" {
		return &v
	}
	return nil
}

// optionalInt64 parses an optional numeric query param; an unparseable value
// is treated as absent rather than an error.
func optionalInt64(r *http.Request, key string) *int64 {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// optionalBool parses an optional "true"/"false" query param. Anything else,
// including an absent or malformed value, is treated as "no filter" (nil).
func optionalBool(r *http.Request, key string) *bool {
	switch strings.TrimSpace(r.URL.Query().Get(key)) {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		return nil
	}
}

// parseBroadcastBody decodes the broadcast request body — a single "text"
// field. Emptiness and length are validated downstream by the broadcast
// service, so all callers map one consistent set of errors.
func parseBroadcastBody(r *http.Request) (string, error) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var body struct {
		Text string `json:"text"`
	}
	if err := dec.Decode(&body); err != nil {
		return "", errors.New("некорректное тело запроса: ожидается поле text")
	}
	return body.Text, nil
}
