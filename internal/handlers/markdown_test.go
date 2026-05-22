package handlers

import (
	"strings"
	"testing"
)

func TestMarkdownToTelegramHTML_PlaceholderDoesNotLeak(t *testing.T) {
	t.Parallel()
	// Regression: when the AI reply contains many inline-code spans the
	// placeholders we used to splice them in had underscores that the
	// italic regex would chew through, leaving raw "PLACEHOLDER10" strings
	// in the final output. This input mimics a real Qwen3 reply that
	// triggered the bug — many adjacent `code` spans in plain prose.
	input := "Нужно написать функцию `PrintNbr`, которая печатает число через `z01.PrintRune`, " +
		"поддерживая отрицательные с `-` и используя рекурсию. Функция `PrintNbr` должна " +
		"обработать отрицательные, потом вызвать `PrintPositiveNum`. `PrintPositiveNum` " +
		"использует рекурсию: считает цифру через `c++`, потом `n/10` для следующей."

	got := markdownToTelegramHTML(input)

	if strings.Contains(got, "PLACEHOLDER") {
		t.Fatalf("placeholder leaked into output: %q", got)
	}
	if strings.ContainsAny(got, "\x00\x01\x02") {
		t.Fatalf("control characters leaked into output: %q", got)
	}
	// Sanity check: <code> tags were produced for each backtick span.
	codeCount := strings.Count(got, "<code>")
	if codeCount < 5 {
		t.Fatalf("expected 5+ <code> tags, got %d in %q", codeCount, got)
	}
}

func TestMarkdownToTelegramHTML_CodeBlockWithLanguage(t *testing.T) {
	t.Parallel()
	input := "Решение:\n```go\nfunc PrintNbr(n int) {\n    z01.PrintRune('-')\n}\n```\nГотово."
	got := markdownToTelegramHTML(input)
	if !strings.Contains(got, `<pre><code class="language-go">`) {
		t.Fatalf("missing language-tagged code block: %q", got)
	}
	if !strings.Contains(got, "</code></pre>") {
		t.Fatalf("missing closing code block: %q", got)
	}
	// Less-than in code should be HTML-escaped.
	input2 := "```go\nif i < 10 {}\n```"
	got2 := markdownToTelegramHTML(input2)
	if !strings.Contains(got2, "i &lt; 10") {
		t.Fatalf("code content not HTML-escaped: %q", got2)
	}
}

func TestMarkdownToTelegramHTML_BoldAndItalic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"bold", "**важно**", "<b>важно</b>"},
		{"italic", "*курсив*", "<i>курсив</i>"},
		{"italic underscore", "_emphasized_", "<i>emphasized</i>"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := markdownToTelegramHTML(tc.in)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("want %q in output, got %q", tc.want, got)
			}
		})
	}
}

func TestIndexToken_NoUnderscoreOrAsterisk(t *testing.T) {
	t.Parallel()
	// Whichever index we pick must produce a token that survives the
	// bold/italic regex passes — i.e. only letters.
	for _, n := range []int{0, 1, 25, 26, 27, 100, 999} {
		tok := indexToken(n)
		if tok == "" {
			t.Fatalf("indexToken(%d) returned empty", n)
		}
		for _, r := range tok {
			if !(r >= 'a' && r <= 'z') {
				t.Fatalf("indexToken(%d) = %q contains non-letter %q", n, tok, r)
			}
		}
	}
}
