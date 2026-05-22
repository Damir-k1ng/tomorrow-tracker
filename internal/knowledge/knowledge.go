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
}

// Match is a search result with the matched Exercise and a relevance score.
// Higher score means stronger match — see scoring rules in Search.
type Match struct {
	Exercise *Exercise
	Score    int
}

// Search finds exercises relevant to the user's free-form question. The
// caller is expected to pass at most a few sentences of natural language.
// Returns at most maxResults matches, ordered by descending score. When
// nothing scores above zero the result is nil — callers treat that as
// "no relevant reference, skip RAG injection".
//
// Scoring is intentionally simple and stable:
//   - +100 when the question contains the canonical lowercase name
//     ("pointone", "ultimatedivmod") — these are unambiguous direct hits.
//   - +50 when the question contains the DisplayName ("PointOne").
//   - +10 for each concept keyword found in the question.
//   - +2 for each significant word from the description found in the
//     question, capped at one credit per word.
//
// Stop-words like "что", "как", "и", "в" are ignored on the description
// pass so that filler doesn't inflate matches.
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

	matches := make([]Match, 0, len(Exercises))
	for i := range Exercises {
		ex := &Exercises[i]
		score := scoreExercise(ex, q, qSet)
		if score > 0 {
			matches = append(matches, Match{Exercise: ex, Score: score})
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

func scoreExercise(ex *Exercise, q string, qSet map[string]struct{}) int {
	score := 0
	if strings.Contains(q, ex.Name) {
		score += 100
	}
	if strings.Contains(q, strings.ToLower(ex.DisplayName)) && ex.DisplayName != ex.Name {
		score += 50
	}
	for _, c := range ex.Concepts {
		if strings.Contains(q, strings.ToLower(c)) {
			score += 10
		}
	}
	for _, w := range tokenize(strings.ToLower(ex.Description)) {
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
