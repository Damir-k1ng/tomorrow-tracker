package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Envelope is the single response shape for every /api/v1 endpoint.
//
//	success:  {"success": true,  "data": ...}
//	error:    {"success": false, "error": "message"}
//
// Exactly one of Data / Error is populated; omitempty keeps the other out.
type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WriteSuccess writes a structured success response with the given status.
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, Envelope{Success: true, Data: data})
}

// WriteError writes a structured error response with the given status.
// The message is user-facing, so keep it generic — never leak internals.
func WriteError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Envelope{Success: false, Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The header is already sent; nothing actionable remains but a log.
		slog.Error("api: encode response failed", slog.String("error", err.Error()))
	}
}
