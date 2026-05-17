package api

import (
	"net/http"
	"strings"
)

// parseOrigins splits a comma-separated CORS origins config into a slice.
// A single "*" entry means "allow any origin".
func parseOrigins(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out
}

// cors is the CORS middleware. It is intentionally minimal: the Mini App is a
// simple token-in-header client (no cookies), so wildcard origins are safe and
// no credentialed-request handling is needed.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowed := s.resolveOrigin(r.Header.Get("Origin")); allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")

		// Preflight: answer and stop before auth / rate-limit run.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// resolveOrigin returns the value to echo in Access-Control-Allow-Origin, or
// "" when the request origin is not permitted.
func (s *Server) resolveOrigin(origin string) string {
	for _, allowed := range s.corsOrigins {
		if allowed == "*" {
			return "*"
		}
		if allowed == origin {
			return origin
		}
	}
	return ""
}
