package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testBotToken = "123456:TEST-bot-token-AbCdEf"

// buildInitData produces a correctly-signed initData string for testing,
// reusing the exact algorithm VerifyInitData expects.
func buildInitData(botToken string, user TelegramUser, authDate time.Time) string {
	return buildInitDataWith(botToken, user, authDate, "")
}

// buildInitDataWith builds a correctly-signed initData, optionally including a
// `signature` field. It mirrors real Telegram: the HMAC `hash` is computed
// over EVERY field except `hash` itself — `signature` included — so a payload
// built here with a signature is signed the way Telegram signs it.
func buildInitDataWith(botToken string, user TelegramUser, authDate time.Time, signature string) string {
	userJSON, _ := json.Marshal(user)
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(authDate.Unix(), 10))
	v.Set("query_id", "AAH123")
	v.Set("user", string(userJSON))
	if signature != "" {
		v.Set("signature", signature)
	}

	// data-check-string: every key except `hash`, sorted, "key=value" by '\n'.
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + "=" + v.Get(k))
	}

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(sb.String()))
	v.Set("hash", hex.EncodeToString(mac.Sum(nil)))

	return v.Encode()
}

func sampleUser() TelegramUser {
	return TelegramUser{ID: 165146312, FirstName: "Damir", Username: "king_traff"}
}

func TestVerifyInitData_ValidPayload(t *testing.T) {
	raw := buildInitData(testBotToken, sampleUser(), time.Now())

	data, err := VerifyInitData(raw, testBotToken, initDataMaxAge)
	if err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
	if data.User.ID != 165146312 || data.User.FirstName != "Damir" {
		t.Fatalf("user not parsed correctly: %+v", data.User)
	}
}

func TestVerifyInitData_TamperedHash(t *testing.T) {
	raw := buildInitData(testBotToken, sampleUser(), time.Now())
	// Flip the hash to an obviously wrong but well-formed value.
	tampered := strings.Replace(raw, "hash=", "hash=deadbeef", 1)

	if _, err := VerifyInitData(tampered, testBotToken, initDataMaxAge); !errors.Is(err, ErrInitDataBadSignature) {
		t.Fatalf("expected ErrInitDataBadSignature, got %v", err)
	}
}

func TestVerifyInitData_WrongBotToken(t *testing.T) {
	raw := buildInitData(testBotToken, sampleUser(), time.Now())

	if _, err := VerifyInitData(raw, "999:WRONG-token", initDataMaxAge); !errors.Is(err, ErrInitDataBadSignature) {
		t.Fatalf("expected ErrInitDataBadSignature for wrong token, got %v", err)
	}
}

func TestVerifyInitData_Expired(t *testing.T) {
	raw := buildInitData(testBotToken, sampleUser(), time.Now().Add(-48*time.Hour))

	if _, err := VerifyInitData(raw, testBotToken, initDataMaxAge); !errors.Is(err, ErrInitDataExpired) {
		t.Fatalf("expected ErrInitDataExpired, got %v", err)
	}
}

func TestVerifyInitData_MissingHash(t *testing.T) {
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("user", `{"id":1}`)

	if _, err := VerifyInitData(v.Encode(), testBotToken, initDataMaxAge); !errors.Is(err, ErrInitDataMissingHash) {
		t.Fatalf("expected ErrInitDataMissingHash, got %v", err)
	}
}

func TestVerifyInitData_WithSignatureField(t *testing.T) {
	// Real Telegram (Bot API 8.0+) sends a `signature` field, and the HMAC
	// `hash` is computed OVER it. A payload signed with `signature` present
	// must verify — this is the exact launch shape of a modern Mini App.
	raw := buildInitDataWith(testBotToken, sampleUser(), time.Now(), "ed25519-third-party-sig")

	if _, err := VerifyInitData(raw, testBotToken, initDataMaxAge); err != nil {
		t.Fatalf("payload with a signature field should verify, got error: %v", err)
	}
}

func TestVerifyInitData_TamperedSignatureFails(t *testing.T) {
	// `signature` is part of the data-check-string, so altering it after
	// signing must invalidate the HMAC `hash`.
	raw := buildInitDataWith(testBotToken, sampleUser(), time.Now(), "original-signature")
	tampered := strings.Replace(raw,
		url.QueryEscape("original-signature"), url.QueryEscape("evil-signature"), 1)

	if _, err := VerifyInitData(tampered, testBotToken, initDataMaxAge); !errors.Is(err, ErrInitDataBadSignature) {
		t.Fatalf("expected ErrInitDataBadSignature for a tampered signature, got %v", err)
	}
}
