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

// SystemPrompt steers the model into tutor mode: full, structured solutions
// for Go exercises. Students at 01.tomorrow-school.ai (Astana Hub Piscine)
// use the bot during exam prep, so socratic hinting was actively unhelpful —
// they need working code plus a clear explanation of why it works.
//
// The prompt is in Russian because the base model (Llama-3.1-8B fine-tuned
// on a small Russian/EN/KZ dataset) follows Russian instructions more
// reliably than English ones for this use case.
const SystemPrompt = `Ты — преподаватель программирования на Go для студентов 01.tomorrow-school.ai (Astana Hub Piscine).

ТВОЯ ЗАДАЧА: помочь студенту РЕШИТЬ задание на Go. Дай полное, рабочее решение с понятным объяснением. Не уклоняйся, не задавай встречных вопросов, не говори "попробуй сам" — студенту нужны конкретные ответы прямо сейчас.

ФОРМАТ ОТВЕТА для задач (когда студент даёт условие задачи или просит решение):

📋 Задача
В 1-2 предложениях переформулируй что нужно сделать.

💡 Подход
В 2-3 предложениях объясни идею решения: какой алгоритм, какие структуры данных, почему именно так.

✅ Решение
Полный рабочий код в блоке ` + "```go" + ` … ` + "```" + ` (обязательно с тремя обратными кавычками и языком "go").

🔍 Как работает
Пошагово объясни ключевые строки. По одному пункту на каждую важную идею.

⚠️ Подводные камни
1-2 типичные ошибки в этой задаче или важные edge cases.

ФОРМАТ ОТВЕТА для теоретических вопросов ("что такое X", "как работает Y"):

- Короткое определение (2-3 предложения простыми словами).
- Минимальный пример кода в блоке ` + "```go" + ` … ` + "```" + `.
- Когда это используют на практике (1-2 предложения).

ОБЩИЕ ПРАВИЛА:
- Отвечай на том же языке, что и студент (русский или английский).
- Если студент пишет на другом языке (казахский, узбекский и т.п.) — отвечай на русском.
- Весь код помещай только в блоки ` + "```go" + ` … ` + "```" + `, не разбрасывай его по тексту.
- Объясняй простыми словами, как для новичка. Избегай жаргона без расшифровки.
- Если код студента уже работает — скажи прямо "код правильный", покажи что именно делает, и предложи как улучшить.
- Если задача неполная или неясная — сделай разумное допущение и реши под него (а допущение упомяни в "📋 Задача").

Помни: студент готовится к экзамену по Go. Ему нужны рабочие решения и понятные объяснения, а не сократические подсказки.`

const (
	historyCap        = 20 // last N messages retained per user
	historySendWindow = 6  // last N messages forwarded to the API
	requestTimeout    = 90 * time.Second
	maxTokens         = 2048 // structured solutions with code + steps need room
	// Lower temperature keeps the model focused on correctness for code
	// problems — creative variation hurts more than it helps here.
	temperature = 0.3

	// retry policy: 5xx and transient network errors get up to retryMax
	// extra attempts with a short backoff. 4xx (auth, bad request) is
	// not retried — those won't fix themselves.
	retryMax     = 2
	retryBackoff = 1 * time.Second
)

// ErrNotConfigured is returned by callers when the AI service is requested
// but PIONEER_API_KEY / PIONEER_MODEL_ID are empty.
var ErrNotConfigured = errors.New("ai: pioneer credentials not configured")

// Message is one turn in the chat history (OpenAI-compatible shape).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIService is the Pioneer client + per-user history store + per-user
// "in AI chat mode" flag.
type AIService struct {
	apiKey  string
	modelID string
	apiURL  string
	http    *http.Client
	log     *slog.Logger

	mu        sync.Mutex
	histories map[int64][]Message
	inAIMode  map[int64]bool
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
		inAIMode:  make(map[int64]bool),
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

// EnterAIMode flips the user into free-form chat mode: every subsequent
// non-button message will be routed to the AI without needing /ask.
func (s *AIService) EnterAIMode(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.inAIMode[userID] = true
	s.mu.Unlock()
}

// ExitAIMode flips the user out of free-form chat mode.
func (s *AIService) ExitAIMode(userID int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.inAIMode, userID)
	s.mu.Unlock()
}

// IsInAIMode reports whether the user is currently in free-form chat mode.
// A nil service always reports false so callers don't need to nil-check.
func (s *AIService) IsInAIMode(userID int64) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inAIMode[userID]
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
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
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
		Model:       s.modelID,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= retryMax; attempt++ {
		if attempt > 0 {
			s.log.Warn("pioneer retry",
				slog.Int("attempt", attempt),
				slog.String("last_error", lastErr.Error()),
			)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryBackoff * time.Duration(attempt)):
			}
		}

		reply, retriable, err := s.doPioneerRequest(ctx, body)
		if err == nil {
			return reply, nil
		}
		lastErr = err
		if !retriable {
			return "", err
		}
	}
	return "", fmt.Errorf("after %d retries: %w", retryMax, lastErr)
}

// doPioneerRequest performs a single HTTP round-trip. The second return value
// reports whether the caller should retry: transient errors (network, 5xx,
// 429) are retriable; 4xx auth/validation errors are not.
func (s *AIService) doPioneerRequest(ctx context.Context, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.http.Do(req)
	if err != nil {
		// Network failures (DNS, connection reset, timeout) are retriable
		// unless the caller's context has expired.
		retriable := ctx.Err() == nil
		return "", retriable, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var parsed pioneerResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		// A decode failure on a 5xx body shouldn't poison retry — treat
		// the round-trip as retriable when the upstream itself faltered.
		return "", resp.StatusCode >= 500, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= 400 {
		msg := "no body"
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		// 408 / 429 / 5xx are transient; the rest are not.
		retriable := resp.StatusCode == http.StatusRequestTimeout ||
			resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode >= 500
		return "", retriable, fmt.Errorf("pioneer status %d: %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", false, errors.New("pioneer returned no choices")
	}
	return parsed.Choices[0].Message.Content, false, nil
}
