// Package knowledge is the bot's RAG-style reference layer. It holds the
// canonical solutions and metadata for every Piscine Go exercise that
// students at 01.tomorrow-school.ai encounter, plus a small search function
// the AI handler uses to inject a ground-truth reference into the prompt
// before the model answers a question.
//
// The data is hand-curated rather than loaded from disk at startup:
//   - the set is small (~40 exercises) and stable;
//   - the embedded form means the binary is self-contained and there is
//     nothing to mount, restore, or wait for at boot;
//   - search runs against in-memory data with no goroutine fan-out, so a
//     /ask call adds well under a millisecond of lookup overhead.
package knowledge

import (
	"sort"
	"strings"
	"unicode"
)

// Exercise is a single Piscine task with its reference solution and the
// metadata the search needs to find it from a free-form student question.
type Exercise struct {
	// Name is the lowercase canonical key — same as the filename in the
	// upstream allprojects repo. Used for direct "помоги с pointone" hits.
	Name string
	// DisplayName is the exported Go identifier shown to the student.
	DisplayName string
	// Signature is the expected function signature for the exercise.
	Signature string
	// Solution is the full reference Go source, including the package
	// declaration so it compiles as-is when pasted into a piscine package.
	Solution string
	// Description is a one- or two-sentence task description in Russian.
	// Students phrase questions in Russian, so the Russian description is
	// what the keyword scorer matches against.
	Description string
	// Concepts are search keywords (Russian + English) that should pull
	// this exercise to the top when mentioned in a free-form question.
	Concepts []string
	// Samples are canonical input → expected-output pairs taken from the
	// subject screenshots. Used for tasks where the grader does byte-level
	// output comparison (visual ASCII-art like QuadA–E, Rectangle, Tetris)
	// — the AI prompt instructs the model to QUOTE these verbatim in the
	// "🧪 Edge cases" section instead of fabricating its own examples.
	// Empty string means "no canonical samples available; the model may
	// generate its own". Format is freeform but typically:
	//
	//   piscine.QuadA(5,3) →
	//   o---o
	//   |   |
	//   o---o
	//
	Samples string
}

// Kind tags a Match with the source registry the hit came from. The AI
// handler uses it to pick between "📌 Упражнение:" and "📚 Концепция Go:"
// framing when rendering the reference block for the model.
type Kind string

const (
	KindExercise Kind = "exercise"
	KindConcept  Kind = "concept"
)

// Match is a search result with the matched item and a relevance score.
// Exactly one of Exercise / Concept is non-nil — pick by Kind. Higher
// Score means stronger match (scoring rules below in Search).
type Match struct {
	Kind     Kind
	Exercise *Exercise // set when Kind == KindExercise
	Concept  *Concept  // set when Kind == KindConcept
	Score    int
}

// Name returns the canonical lowercase key of the matched item.
func (m Match) Name() string {
	if m.Exercise != nil {
		return m.Exercise.Name
	}
	if m.Concept != nil {
		return m.Concept.Name
	}
	return ""
}

// Search finds exercises and concepts relevant to the user's free-form
// question. The caller is expected to pass at most a few sentences of
// natural language. Returns at most maxResults matches, ordered by
// descending score. When nothing scores above zero the result is nil —
// callers treat that as "no relevant reference, skip RAG injection".
//
// Scoring is intentionally simple and stable:
//   - +100 when the question contains the canonical lowercase name
//     ("pointone", "goroutines") — these are unambiguous direct hits.
//   - +50 when the question contains the DisplayName ("PointOne",
//     "Goroutines") and it differs from the canonical name.
//   - +10 for each concept keyword found in the question.
//   - +2 for each significant word from the description found in the
//     question, capped at one credit per word.
//
// Stop-words like "что", "как", "и", "в" are ignored on the description
// pass so that filler doesn't inflate matches. Exercises and concepts
// compete in the same ranking so a strong concept hit can outrank a
// weak exercise hit and vice versa.
func Search(question string, maxResults int) []Match {
	if maxResults <= 0 {
		maxResults = 3
	}
	q := normalize(question)
	if q == "" {
		return nil
	}
	qWords := tokenize(q)
	qSet := make(map[string]struct{}, len(qWords))
	for _, w := range qWords {
		qSet[w] = struct{}{}
	}

	matches := make([]Match, 0, len(Exercises)+len(Concepts))
	for i := range Exercises {
		ex := &Exercises[i]
		score := scoreEntry(ex.Name, ex.DisplayName, ex.Description, ex.Concepts, q, qSet)
		if score > 0 {
			matches = append(matches, Match{Kind: KindExercise, Exercise: ex, Score: score})
		}
	}
	for i := range Concepts {
		c := &Concepts[i]
		score := scoreEntry(c.Name, c.DisplayName, c.Description, c.Concepts, q, qSet)
		if score > 0 {
			matches = append(matches, Match{Kind: KindConcept, Concept: c, Score: score})
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}
	return matches
}

// Ambiguity reports that a free-form question maps to several plausible
// exercises with no clear winner — typically the student named a concept
// shared by multiple tasks ("прямоугольник", "линейный список") without
// picking a specific exercise. The AI service uses this signal to ask
// the student to disambiguate instead of guessing and producing the
// wrong canonical solution.
type Ambiguity struct {
	// Candidates are the exercises tied at the top score, in registry
	// order (which matches "QuadA, QuadB, ...Ee" rather than some
	// surprising rearrangement).
	Candidates []*Exercise
	// Score is the shared top score across all Candidates. Useful for
	// logging and for the caller to decide whether the tie is even
	// strong enough to warrant a clarification prompt.
	Score int
}

// Disambiguate runs Search and reports whether the result is genuinely
// ambiguous — i.e. multiple Piscine exercises tie for the top score with
// no direct-name winner. Returns nil when:
//
//   - The query has a direct exercise-name hit (top score ≥ 100). One
//     candidate dominates; asking would be annoying.
//   - The top match is weak (score below ambiguityFloor). Most likely an
//     unrelated query — let the model answer from its own weights.
//   - Only one exercise reaches the top score (no tie).
//
// When non-nil, the caller should present Candidates to the student and
// skip the LLM round-trip. Disambiguation answers are never cached: the
// next question depends on which option the student picks.
func Disambiguate(question string) *Ambiguity {
	matches := Search(question, 5)
	if len(matches) < 2 {
		return nil
	}
	top := matches[0].Score
	// Direct name match — no ambiguity, one exercise clearly wins.
	if top >= directNameScore {
		return nil
	}
	// Tie must be strong enough to be worth asking. Concepts-only hits
	// (single +10 from a shared keyword) are exactly the case where
	// asking helps.
	if top < ambiguityFloor {
		return nil
	}
	var tied []*Exercise
	for _, m := range matches {
		if m.Score != top {
			break // matches are sorted descending; first miss ends the tie
		}
		if m.Kind == KindExercise && m.Exercise != nil {
			tied = append(tied, m.Exercise)
		}
	}
	if len(tied) < 2 {
		return nil
	}
	return &Ambiguity{Candidates: tied, Score: top}
}

const (
	// directNameScore is the score awarded by scoreEntry when the
	// question contains the canonical lowercase exercise name. Anything
	// at or above this is treated as a direct hit — never ambiguous.
	directNameScore = 100
	// ambiguityFloor is the minimum top score that justifies pausing
	// the pipeline to ask "which one?". Below this we'd interrupt the
	// student over a coincidence, which is worse than answering best-guess.
	ambiguityFloor = 10
)

// FindByName returns the exercise with the given canonical name (e.g.
// "pointone"), or nil if it's not in the registry. Useful for direct lookups
// that bypass the scorer when the caller already knows the exercise key.
func FindByName(name string) *Exercise {
	n := strings.ToLower(strings.TrimSpace(name))
	for i := range Exercises {
		if Exercises[i].Name == n {
			return &Exercises[i]
		}
	}
	return nil
}

// scoreEntry is the shared scoring function used for both exercises and
// concepts. Pulled out of the per-type loop so both registries are ranked
// by the exact same rules, which keeps the cross-type ordering predictable.
//
// Important: name and displayName match on WORD BOUNDARIES, not substring.
// Otherwise the short name "point" would steal "помоги решить pointone"
// from the more specific "pointone" entry. Concepts still match on
// substring because they're often multi-word phrases ("простое число").
func scoreEntry(name, displayName, description string, concepts []string, q string, qSet map[string]struct{}) int {
	score := 0
	if _, ok := qSet[name]; ok {
		score += 100
	}
	displayLower := strings.ToLower(displayName)
	if displayLower != name {
		if _, ok := qSet[displayLower]; ok {
			score += 50
		}
	}
	for _, c := range concepts {
		if strings.Contains(q, strings.ToLower(c)) {
			score += 10
		}
	}
	for _, w := range tokenize(strings.ToLower(description)) {
		if len(w) < 4 || isStopWord(w) {
			continue
		}
		if _, ok := qSet[w]; ok {
			score += 2
		}
	}
	return score
}

// normalize lowercases and strips punctuation so that "Хочу решить PointOne!"
// matches the same way as "хочу решить pointone".
func normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == ' ', r == '\n':
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func tokenize(s string) []string {
	return strings.Fields(s)
}

// isStopWord filters out the common short Russian and English fillers that
// otherwise inflate matches against every exercise that mentions them.
func isStopWord(w string) bool {
	switch w {
	case "что", "как", "это", "для", "если", "или", "при", "так", "там",
		"вот", "уже", "ещё", "еще", "тоже", "когда", "тогда",
		"the", "and", "for", "with", "from", "this", "that", "into",
		"return", "func", "package", "piscine":
		return true
	}
	return false
}
