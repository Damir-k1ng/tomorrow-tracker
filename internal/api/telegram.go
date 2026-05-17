// Package api hosts the HTTP layer for the future Telegram Mini App and Admin
// Panel. The Telegram bot runs independently; this server only adds a v1 API
// surface alongside it.
package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// initDataMaxAge bounds how old a Telegram initData payload may be. Telegram's
// own examples use 24h; anything older is rejected as a stale/replayed login.
const initDataMaxAge = 24 * time.Hour

// Verification errors. They are deliberately coarse so HTTP handlers can map
// every failure to a single generic 401 without leaking which check failed.
var (
	ErrInitDataMissingHash  = errors.New("init data: missing hash")
	ErrInitDataMissingDate  = errors.New("init data: missing auth_date")
	ErrInitDataBadSignature = errors.New("init data: signature mismatch")
	ErrInitDataExpired      = errors.New("init data: expired")
	ErrInitDataMissingUser  = errors.New("init data: missing or invalid user")
)

// TelegramUser is the subset of the Telegram WebApp user object we consume.
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`
}

// InitData is the verified result of a Telegram Mini App initData payload.
type InitData struct {
	User     TelegramUser
	AuthDate time.Time
	QueryID  string
}

// VerifyInitData validates a Telegram Mini App initData string using the
// official WebApp verification flow:
//
//  1. Parse the URL-encoded payload.
//  2. Build the data-check-string: all fields except `hash` and `signature`,
//     sorted by key, joined as "key=value" with '\n'.
//  3. secret_key  = HMAC_SHA256(key="WebAppData", data=bot_token)
//  4. expected    = HMAC_SHA256(key=secret_key, data=data_check_string)
//  5. Constant-time compare expected (hex) against the supplied `hash`.
//  6. Reject payloads older than maxAge.
//
// The `signature` field (Telegram's separate Ed25519 third-party signature) is
// excluded from the data-check-string, matching Telegram's reference behavior.
//
// This is security-critical: an invalid signature or expired payload yields an
// error and the caller MUST treat the request as unauthenticated.
func VerifyInitData(rawInitData, botToken string, maxAge time.Duration) (*InitData, error) {
	values, err := url.ParseQuery(rawInitData)
	if err != nil {
		return nil, fmt.Errorf("init data: parse: %w", err)
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, ErrInitDataMissingHash
	}

	dataCheckString := buildDataCheckString(values)

	// Step 3: derive the secret key from the bot token.
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	secretKey := secret.Sum(nil)

	// Step 4: compute the expected hash over the data-check-string.
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac.Sum(nil))

	// Step 5: constant-time comparison defeats timing attacks.
	if !hmac.Equal([]byte(expected), []byte(hash)) {
		return nil, ErrInitDataBadSignature
	}

	// Step 6: the signature is valid, so auth_date is now trustworthy.
	authDateStr := values.Get("auth_date")
	if authDateStr == "" {
		return nil, ErrInitDataMissingDate
	}
	authUnix, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: bad auth_date", ErrInitDataMissingDate)
	}
	authDate := time.Unix(authUnix, 0)
	if maxAge > 0 && time.Since(authDate) > maxAge {
		return nil, ErrInitDataExpired
	}

	userJSON := values.Get("user")
	if userJSON == "" {
		return nil, ErrInitDataMissingUser
	}
	var user TelegramUser
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInitDataMissingUser, err)
	}
	if user.ID == 0 {
		return nil, ErrInitDataMissingUser
	}

	return &InitData{User: user, AuthDate: authDate, QueryID: values.Get("query_id")}, nil
}

// buildDataCheckString assembles the canonical string Telegram signs: every
// field except `hash` and `signature`, sorted by key, "key=value" per line.
func buildDataCheckString(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "hash" || k == "signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(values.Get(k))
	}
	return sb.String()
}
