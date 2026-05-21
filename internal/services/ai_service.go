// Package services — AIService wraps the Pioneer chat-completions API and
// keeps a short in-memory dialog history per Telegram user.
//
// History is intentionally process-local: an in-memory map is enough for the
// MVP and survives the per-request lifecycle. A bot restart wipes context,
// which matches the /clear UX and avoids leaking student code to disk.
package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// userAgent is sent on every Pioneer call. Some CDNs (CloudFront in particular)
// treat anonymous traffic as bot-like and apply tighter rate limits.
const userAgent = "tomorrow-tracker-bot/1.0 (+https://github.com/Damir-k1ng/tomorrow-tracker)"

// pioneerHTTPClient builds the HTTP client used to call Pioneer. Two
// production-only quirks live here:
//
//  1. IPv4 only. Railway containers advertise IPv6 connectivity, but the
//     Railway → AWS CloudFront IPv6 path silently stalls on POST bodies.
//     Forcing "tcp4" makes the dialer skip IPv6 entirely.
//  2. HTTP/1.1 only. The default Go HTTP/2 client occasionally hangs while
//     awaiting headers from CloudFront-fronted endpoints. Disabling h2 keeps
//     us on a code path curl also uses successfully.
func pioneerHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
		// Empty (non-nil) TLSNextProto disables HTTP/2 even when the server
		// advertises h2 via ALPN — see https://pkg.go.dev/net/http#Transport.
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	return &http.Client{
		Timeout:   requestTimeout,
		Transport: tr,
	}
}

// SystemPrompt steers the model into mentor mode: hints, not answers.
const SystemPrompt = `You are a helpful programming mentor for students at 01.tomorrow-school.ai (Astana Hub Piscine).
You help students learn Go programming by analyzing their code, identifying bugs, explaining errors clearly, and giving hints.
You are encouraging — guide students to find the answer themselves, don't just give it away.
Answer in the same language the student uses (Russian, English, or Kazakh).
When analyzing code: 1) Identify the bug, 2) Explain WHY it's wrong, 3) Give a hint toward the fix.`

const (
	historyCap        = 20 // last N messages retained per user
	historySendWindow = 6  // last N messages forwarded to the API
	requestTimeout    = 60 * time.Second
	maxTokens         = 1024
)

// ErrNotConfigured is returned by callers when the AI service is requested
// but PIONEER_API_KEY / PIONEER_MODEL_ID are empty.
var ErrNotConfigured = errors.New("ai: pioneer credentials not configured")

// Message is one turn in the chat history (OpenAI-compatible shape).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIService is the Pioneer client + per-user history store.
type AIService struct {
	apiKey  string
	modelID string
	apiURL  string
	http    *http.Client
	log     *slog.Logger

	mu        sync.Mutex
	histories map[int64][]Message
}

// NewAIService constructs the service. Returns nil if credentials are missing
// so the handler layer can show a graceful "AI disabled" message.
func NewAIService(apiKey, modelID, apiURL string, log *slog.Logger) *AIService {
	if apiKey == "" || modelID == "" {
		return nil
	}
	return &AIService{
		apiKey:    apiKey,
		modelID:   modelID,
		apiURL:    apiURL,
		http:      pioneerHTTPClient(),
		log:       log,
		histories: make(map[int64][]Message),
	}
}

// Ask sends a user question to Pioneer and returns the assistant reply.
// History is mutated only on a successful round-trip — failed calls leave the
// conversation state untouched so the user can retry without duplicated turns.
func (s *AIService) Ask(ctx context.Context, userID int64, question string) (string, error) {
	if s == nil {
		return "", ErrNotConfigured
	}

	prior := s.snapshotHistory(userID)
	messages := buildMessages(prior, question)

	reply, err := s.callPioneer(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("pioneer call: %w", err)
	}

	s.appendTurn(userID, question, reply)
	return reply, nil
}

// Clear drops the dialog history for a single user.
func (s *AIService) Clear(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.histories, userID)
	s.mu.Unlock()
}

// snapshotHistory returns a copy of the user's history so the caller can read
// without holding the mutex across an HTTP round-trip.
func (s *AIService) snapshotHistory(userID int64) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.histories[userID]
	if len(src) == 0 {
		return nil
	}
	out := make([]Message, len(src))
	copy(out, src)
	return out
}

// appendTurn records both sides of one exchange and trims to historyCap.
func (s *AIService) appendTurn(userID int64, question, answer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := append(s.histories[userID],
		Message{Role: "user", Content: question},
		Message{Role: "assistant", Content: answer},
	)
	if len(h) > historyCap {
		h = h[len(h)-historyCap:]
	}
	s.histories[userID] = h
}

// buildMessages prepends the system prompt and limits the forwarded window so
// the model receives recent context without blowing the token budget.
func buildMessages(prior []Message, question string) []Message {
	window := prior
	if len(window) > historySendWindow {
		window = window[len(window)-historySendWindow:]
	}
	msgs := make([]Message, 0, len(window)+2)
	msgs = append(msgs, Message{Role: "system", Content: SystemPrompt})
	msgs = append(msgs, window...)
	msgs = append(msgs, Message{Role: "user", Content: question})
	return msgs
}

// pioneerRequest mirrors the OpenAI-compatible chat-completions schema.
type pioneerRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type pioneerResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *AIService) callPioneer(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(pioneerRequest{
		Model:     s.modelID,
		Messages:  messages,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var parsed pioneerResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= 400 {
		msg := "no body"
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("pioneer status %d: %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("pioneer returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
