package knowledge

import (
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		query       string
		wantTopName string // exercise name expected at the top of results
		wantNoHits  bool   // true if Search should return nil
	}{
		{
			name:        "direct exercise name match",
			query:       "помоги решить pointone",
			wantTopName: "pointone",
		},
		{
			name:        "display name in query",
			query:       "как написать функцию PointOne",
			wantTopName: "pointone",
		},
		{
			name:        "concept keyword surfaces specific exercise",
			query:       "что такое тройной указатель",
			wantTopName: "ultimatepointone",
		},
		{
			name:        "swap by russian description",
			query:       "поменять местами значения двух int",
			wantTopName: "swap",
		},
		{
			name:        "fibonacci concept",
			query:       "напиши фибоначчи",
			wantTopName: "fibonacci",
		},
		{
			name:        "prime concept",
			query:       "проверить простое число",
			wantTopName: "isprime",
		},
		{
			name:        "quada direct name",
			query:       "помоги решить quada",
			wantTopName: "quada",
		},
		{
			name:        "quadb display name",
			query:       "как написать QuadB",
			wantTopName: "quadb",
		},
		{
			name:        "quadc display name",
			query:       "QuadC решение",
			wantTopName: "quadc",
		},
		{
			name:        "quadd display name",
			query:       "QuadD помоги",
			wantTopName: "quadd",
		},
		{
			name:        "quade display name",
			query:       "что вернёт QuadE",
			wantTopName: "quade",
		},
		{
			name:       "unrelated query returns nothing",
			query:      "расскажи про погоду в москве",
			wantNoHits: true,
		},
		{
			name:       "empty query is safe",
			query:      "",
			wantNoHits: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Search(tc.query, 3)
			if tc.wantNoHits {
				if got != nil {
					t.Fatalf("expected no hits, got %d: %+v", len(got), got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("expected at least one hit for %q, got none", tc.query)
			}
			if got[0].Exercise.Name != tc.wantTopName {
				t.Fatalf("for %q want top=%q, got %q (score=%d). all results: %+v",
					tc.query, tc.wantTopName, got[0].Exercise.Name, got[0].Score, got)
			}
		})
	}
}

func TestDisambiguate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		query        string
		wantAmbig    bool
		wantNames    []string // expected DisplayNames in candidates (any order, but all must be present)
		wantMinScore int      // 0 = no lower bound
	}{
		{
			name:      "direct name returns nil — QuadA wins outright",
			query:     "помоги решить QuadA",
			wantAmbig: false,
		},
		{
			name:      "rectangle concept ties across all 5 Quads",
			query:     "напиши прямоугольник",
			wantAmbig: true,
			wantNames: []string{"QuadA", "QuadB", "QuadC", "QuadD", "QuadE"},
		},
		{
			name:      "ramka concept also ties Quads",
			query:     "нарисуй рамку",
			wantAmbig: true,
			wantNames: []string{"QuadA", "QuadB", "QuadC", "QuadD", "QuadE"},
		},
		{
			name:      "unrelated query — no ambiguity, no hits",
			query:     "погода в москве",
			wantAmbig: false,
		},
		{
			name:      "empty query — safe nil",
			query:     "",
			wantAmbig: false,
		},
		{
			name:      "single quad keyword still hits one exercise distinctly",
			query:     "помоги с quadc",
			wantAmbig: false, // direct name match overrides
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Disambiguate(tc.query)
			if !tc.wantAmbig {
				if got != nil {
					names := make([]string, len(got.Candidates))
					for i, ex := range got.Candidates {
						names[i] = ex.DisplayName
					}
					t.Fatalf("expected nil (no ambiguity) for %q, got candidates=%v score=%d", tc.query, names, got.Score)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected ambiguity for %q, got nil", tc.query)
			}
			if len(got.Candidates) < 2 {
				t.Fatalf("ambiguity must have ≥2 candidates, got %d", len(got.Candidates))
			}
			seen := make(map[string]bool, len(got.Candidates))
			for _, ex := range got.Candidates {
				seen[ex.DisplayName] = true
			}
			for _, want := range tc.wantNames {
				if !seen[want] {
					gotNames := make([]string, 0, len(got.Candidates))
					for _, ex := range got.Candidates {
						gotNames = append(gotNames, ex.DisplayName)
					}
					t.Errorf("candidates missing %q; got %v", want, gotNames)
				}
			}
		})
	}
}

func TestFindByName(t *testing.T) {
	t.Parallel()
	if ex := FindByName("pointone"); ex == nil || ex.DisplayName != "PointOne" {
		t.Fatalf("FindByName(pointone) = %v", ex)
	}
	if ex := FindByName("POINTONE"); ex == nil {
		t.Fatalf("FindByName should be case-insensitive")
	}
	if ex := FindByName("nonexistent"); ex != nil {
		t.Fatalf("FindByName(nonexistent) want nil, got %v", ex)
	}
}

// TestQuadSolutionsInvariants guards the QuadA–E reference solutions
// against regressions that previously caused the bot to hallucinate `*`
// or `fmt.Print*` in visual ASCII-art tasks. Each canonical solution
// MUST:
//   - declare package piscine (library form, not standalone main);
//   - import only github.com/01-edu/z01 (no fmt — Piscine grader rejects it
//     on these tasks);
//   - contain the exact runes that the subject's expected output uses
//     (subject diff is byte-level — replacing `o---o` with `*****` is an
//     instant grader FAIL).
//
// If any of these assertions starts failing, do NOT relax the test —
// the bot just lost a regression guard the prompt cannot enforce alone.
func TestQuadSolutionsInvariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		runeChars   []rune   // PrintRune literals expected in the body
		sampleLines []string // byte-exact lines that MUST appear in Samples
	}{
		{
			name:      "quada",
			runeChars: []rune{'o', '-', '|', ' ', '\n'},
			sampleLines: []string{
				"o---o",
				"|   |",
			},
		},
		{
			name:      "quadb",
			runeChars: []rune{'/', '\\', '*', ' ', '\n'},
			sampleLines: []string{
				`/***\`,
				`\***/`,
			},
		},
		{
			name:      "quadc",
			runeChars: []rune{'A', 'B', 'C', ' ', '\n'},
			sampleLines: []string{
				"ABBBA",
				"CBBBC",
			},
		},
		{
			name:      "quadd",
			runeChars: []rune{'A', 'B', 'C', ' ', '\n'},
			sampleLines: []string{
				"ABBBC",
			},
		},
		{
			name:      "quade",
			runeChars: []rune{'A', 'B', 'C', ' ', '\n'},
			sampleLines: []string{
				"ABBBC",
				"CBBBA",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := FindByName(tc.name)
			if ex == nil {
				t.Fatalf("FindByName(%q) returned nil — entry missing from extraExercises", tc.name)
			}
			src := ex.Solution

			if !strings.Contains(src, "package piscine") {
				t.Errorf("%s: solution must declare 'package piscine' (library form), got:\n%s", tc.name, src)
			}
			if !strings.Contains(src, `"github.com/01-edu/z01"`) {
				t.Errorf("%s: solution must import github.com/01-edu/z01", tc.name)
			}
			// fmt is forbidden by Piscine grader for these visual tasks.
			// Use word-boundary check so "format" in a comment wouldn't trip,
			// but "fmt." (the qualified call) absolutely cannot appear.
			if strings.Contains(src, `"fmt"`) || strings.Contains(src, "fmt.") {
				t.Errorf("%s: solution must NOT import or call fmt (Piscine forbids it on this task)", tc.name)
			}
			if !strings.Contains(src, "z01.PrintRune") {
				t.Errorf("%s: solution must emit output via z01.PrintRune", tc.name)
			}
			// Defensive guard against accidental empty-output for non-positive
			// dimensions — subject explicitly says "otherwise, print nothing".
			if !strings.Contains(src, "x <= 0") || !strings.Contains(src, "y <= 0") {
				t.Errorf("%s: solution must guard against x<=0 || y<=0", tc.name)
			}

			for _, r := range tc.runeChars {
				lit := runeLiteral(r)
				if !strings.Contains(src, "z01.PrintRune("+lit+")") {
					t.Errorf("%s: solution must call z01.PrintRune(%s) for rune %q", tc.name, lit, r)
				}
			}

			// Samples are mandatory for visual ASCII tasks — without them
			// the model fabricates plausible-but-wrong outputs in the
			// "🧪 Edge cases" section (real bug observed in production).
			if ex.Samples == "" {
				t.Fatalf("%s: Samples must be populated (byte-exact subject outputs) to prevent hallucinated edge cases", tc.name)
			}
			for _, line := range tc.sampleLines {
				if !strings.Contains(ex.Samples, line) {
					t.Errorf("%s: Samples missing canonical line %q\nGot Samples:\n%s", tc.name, line, ex.Samples)
				}
			}
		})
	}
}

// runeLiteral renders a rune as its Go source literal, e.g. 'A', '\\', '\n'.
// Used by TestQuadSolutionsInvariants to assert PrintRune calls exist.
func runeLiteral(r rune) string {
	switch r {
	case '\n':
		return `'\n'`
	case '\\':
		return `'\\'`
	default:
		return "'" + string(r) + "'"
	}
}

func TestExercisesIntegrity(t *testing.T) {
	t.Parallel()
	if len(Exercises) < 30 {
		t.Fatalf("expected at least 30 exercises, got %d", len(Exercises))
	}
	seen := make(map[string]bool)
	for _, ex := range Exercises {
		if ex.Name == "" || ex.DisplayName == "" {
			t.Errorf("exercise has empty name fields: %+v", ex)
		}
		if seen[ex.Name] {
			t.Errorf("duplicate exercise name: %q", ex.Name)
		}
		seen[ex.Name] = true
		if ex.Signature == "" {
			t.Errorf("%s: missing signature", ex.Name)
		}
		if ex.Solution == "" {
			t.Errorf("%s: missing solution", ex.Name)
		}
		if ex.Description == "" {
			t.Errorf("%s: missing description", ex.Name)
		}
	}
}
