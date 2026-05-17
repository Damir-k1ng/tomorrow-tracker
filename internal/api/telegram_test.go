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
	userJSON, _ := json.Marshal(user)
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(authDate.Unix(), 10))
	v.Set("query_id", "AAH123")
	v.Set("user", string(userJSON))

	// data-check-string: all keys sorted, "key=value" joined by '\n'.
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

func TestVerifyInitData_SignatureFieldExcludedFromCheck(t *testing.T) {
	// A `signature` field must not break HMAC validation — it is excluded
	// from the data-check-string. Appending it to an otherwise valid payload
	// should still verify.
	raw := buildInitData(testBotToken, sampleUser(), time.Now())
	withSig := raw + "&signature=" + url.QueryEscape("ed25519-third-party-sig")

	if _, err := VerifyInitData(withSig, testBotToken, initDataMaxAge); err != nil {
		t.Fatalf("signature field should be ignored, got error: %v", err)
	}
}
