package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// reqInfo carries request-scoped diagnostics that downstream middleware fills
// in — currently the resolved user ID. The access-log middleware runs
// outermost and therefore never sees the request context the inner handlers
// receive (each middleware passes a *derived* request down). A shared pointer
// in the context is the bridge: accessLog creates it, requireTelegramAuth
// mutates it, accessLog reads it back after the handler returns.
type reqInfo struct {
	userID int64
}

const reqInfoContextKey contextKey = "api.reqinfo"

// reqInfoFromContext returns the request-scoped diagnostics holder, or nil when
// the request did not pass through accessLog (e.g. in narrow unit tests).
func reqInfoFromContext(ctx context.Context) *reqInfo {
	ri, _ := ctx.Value(reqInfoContextKey).(*reqInfo)
	return ri
}

// statusRecorder wraps http.ResponseWriter to remember the status code and the
// byte count, so the access log can report both. A handler that writes a body
// without an explicit WriteHeader implicitly produces 200 — Write() mirrors
// that so the logged status is never a misleading zero.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (sr *statusRecorder) WriteHeader(code int) {
	if sr.status == 0 {
		sr.status = code
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	n, err := sr.ResponseWriter.Write(b)
	sr.bytes += n
	return n, err
}

// accessLog emits one structured line per /api request: method, path, status,
// latency, client IP and — once authentication has run — the user ID. It is
// the observability the API was missing entirely: without it a 429 storm or a
// slow query is invisible in production logs. It is wired as the OUTERMOST
// middleware of every /api group so it still records requests that recoverPanic
// turns into a 500.
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ri := &reqInfo{}
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), reqInfoContextKey, ri)))

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		attrs := []any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("took", time.Since(start)),
			slog.String("ip", clientIP(r)),
		}
		if ri.userID != 0 {
			attrs = append(attrs, slog.Int64("user_id", ri.userID))
		}

		// Status drives the level so warnings/errors surface without noise:
		// 5xx → error, 4xx (including 429) → warn, everything else → info.
		level := slog.LevelInfo
		switch {
		case rec.status >= 500:
			level = slog.LevelError
		case rec.status >= 400:
			level = slog.LevelWarn
		}
		s.log.Log(context.Background(), level, "api request", attrs...)
	})
}
