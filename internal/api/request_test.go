package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getReq(target string) *http.Request {
	return httptest.NewRequest(http.MethodGet, target, nil)
}

func TestParsePagination_Defaults(t *testing.T) {
	pg, err := parsePagination(getReq("/x"))
	if err != nil {
		t.Fatal(err)
	}
	if pg.Page != 1 || pg.Limit != defaultPageLimit || pg.Offset != 0 {
		t.Fatalf("unexpected defaults: %+v", pg)
	}
}

func TestParsePagination_OffsetAndCap(t *testing.T) {
	pg, err := parsePagination(getReq("/x?page=3&limit=999"))
	if err != nil {
		t.Fatal(err)
	}
	if pg.Limit != maxPageLimit {
		t.Errorf("limit not capped: got %d want %d", pg.Limit, maxPageLimit)
	}
	if pg.Offset != (3-1)*maxPageLimit {
		t.Errorf("offset wrong: got %d", pg.Offset)
	}
}

func TestParsePagination_Invalid(t *testing.T) {
	for _, target := range []string{"/x?page=0", "/x?page=-1", "/x?page=abc", "/x?limit=0", "/x?limit=xx"} {
		if _, err := parsePagination(getReq(target)); err == nil {
			t.Errorf("expected error for %q", target)
		}
	}
}

func TestParseSort_WhitelistAndDirection(t *testing.T) {
	cases := []struct {
		query    string
		wantCol  string
		wantDesc bool
	}{
		{"/x", "created_at", true},                          // default
		{"/x?sort=current_streak", "current_streak", false}, // explicit asc
		{"/x?sort=-best_streak", "best_streak", true},       // explicit desc
		{"/x?sort=evil_column", "created_at", true},         // unknown key → default
		{"/x?sort=-DROP", "created_at", true},               // injection attempt → default
	}
	for _, c := range cases {
		col, desc := parseSort(getReq(c.query), userSortColumns, "created_at", true)
		if col != c.wantCol || desc != c.wantDesc {
			t.Errorf("%s: got (%q,%v) want (%q,%v)", c.query, col, desc, c.wantCol, c.wantDesc)
		}
	}
}

func TestParseExportRange(t *testing.T) {
	// Missing params.
	if _, err := parseExportRange(getReq("/x?from=2026-01-01")); err == nil {
		t.Error("expected error when `to` missing")
	}
	// Bad format.
	if _, err := parseExportRange(getReq("/x?from=01-01-2026&to=2026-02-01")); err == nil {
		t.Error("expected error for bad date format")
	}
	// Reversed range.
	if _, err := parseExportRange(getReq("/x?from=2026-02-01&to=2026-01-01")); err == nil {
		t.Error("expected error when to < from")
	}
	// Over the max span.
	if _, err := parseExportRange(getReq("/x?from=2024-01-01&to=2026-01-01")); err == nil {
		t.Error("expected error for oversized range")
	}
	// Valid: `to` is made exclusive (end date + 1 day).
	rng, err := parseExportRange(getReq("/x?from=2026-01-01&to=2026-01-31"))
	if err != nil {
		t.Fatal(err)
	}
	if rng.To.Day() != 1 || rng.To.Month() != 2 {
		t.Errorf("expected To extended to 2026-02-01, got %v", rng.To)
	}
}

func patchReq(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))
	return r
}

func TestParseSessionPatch_Valid(t *testing.T) {
	p, err := parseSessionPatch(patchReq(`{"duration_minutes":95,"reason":"double tap"}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.DurationMinutes == nil || *p.DurationMinutes != 95 || p.Reason != "double tap" {
		t.Fatalf("unexpected parse: %+v", p)
	}
	if p.IsValid != nil || p.SetFlags {
		t.Errorf("absent fields should stay unset: %+v", p)
	}
}

func TestParseSessionPatch_PartialFields(t *testing.T) {
	// is_valid alone — no duration restated.
	p, err := parseSessionPatch(patchReq(`{"is_valid":false,"reason":"чит"}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.DurationMinutes != nil || p.IsValid == nil || *p.IsValid {
		t.Fatalf("expected only is_valid=false set: %+v", p)
	}

	// anti_cheat_flags alone — deduplicated and sorted into canonical JSON.
	p, err = parseSessionPatch(patchReq(
		`{"anti_cheat_flags":["rapid_restarts","manual_review","rapid_restarts"],"reason":"r"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !p.SetFlags || string(p.AntiCheatFlags) != `["manual_review","rapid_restarts"]` {
		t.Fatalf("unexpected canonical flags: %q", p.AntiCheatFlags)
	}

	// Empty array clears flags to the canonical "[]" — never JSON null.
	p, err = parseSessionPatch(patchReq(`{"anti_cheat_flags":[],"reason":"r"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !p.SetFlags || string(p.AntiCheatFlags) != `[]` {
		t.Fatalf("expected canonical empty array, got %q", p.AntiCheatFlags)
	}
}

func TestParseSessionPatch_Rejections(t *testing.T) {
	cases := map[string]string{
		"no editable field":  `{"reason":"x"}`,
		"missing reason":     `{"duration_minutes":10}`,
		"blank reason":       `{"duration_minutes":10,"reason":"   "}`,
		"negative duration":  `{"duration_minutes":-1,"reason":"x"}`,
		"over cap duration":  `{"duration_minutes":721,"reason":"x"}`,
		"unknown field":      `{"duration_minutes":10,"reason":"x","telegram_id":999}`,
		"malformed json":     `{not json`,
		"unknown flag":       `{"anti_cheat_flags":["wallhack"],"reason":"x"}`,
		"flags not an array": `{"anti_cheat_flags":"manual_review","reason":"x"}`,
		"flags wrong elem":   `{"anti_cheat_flags":[123],"reason":"x"}`,
	}
	for name, body := range cases {
		if _, err := parseSessionPatch(patchReq(body)); err == nil {
			t.Errorf("%s: expected rejection, got nil", name)
		}
	}
}

func TestParsePathID(t *testing.T) {
	ok := httptest.NewRequest(http.MethodGet, "/x", nil)
	ok.SetPathValue("id", "42")
	if id, err := parsePathID(ok); err != nil || id != 42 {
		t.Fatalf("valid id: got %d err=%v", id, err)
	}
	for _, bad := range []string{"0", "-3", "abc", ""} {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.SetPathValue("id", bad)
		if _, err := parsePathID(r); err == nil {
			t.Errorf("expected error for id %q", bad)
		}
	}
}
