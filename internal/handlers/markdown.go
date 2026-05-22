package handlers

import (
	"regexp"
	"strings"
)

// Telegram HTML parse_mode supports a tiny subset: <b>, <i>, <u>, <s>,
// <code>, <pre>, <pre><code class="language-…">, <a href>. Anything else
// raises a Bad Request. This file converts the model's loose Markdown to
// that subset, preserving fenced code blocks and inline code while escaping
// the rest of the text.

var (
	// Fenced code block: ```lang\n…\n``` (lang is optional). Non-greedy.
	reCodeBlock = regexp.MustCompile("(?s)```([A-Za-z0-9_+-]*)\\n?(.*?)```")
	// Inline code: `…`. Single backticks only — must not overlap fenced blocks.
	reInlineCode = regexp.MustCompile("`([^`\\n]+?)`")
	// Bold: **…** or __…__.
	reBold = regexp.MustCompile(`\*\*([^*\n]+?)\*\*|__([^_\n]+?)__`)
	// Italic: *…* or _…_. Must not match the bold delimiters above; handled
	// by running this pass after the bold pass on the remaining text.
	reItalic = regexp.MustCompile(`\*([^*\n]+?)\*|_([^_\n]+?)_`)
)

// markdownToTelegramHTML converts the model's Markdown into the Telegram
// HTML dialect. Order matters: extract fenced blocks → inline code → escape
// HTML in the rest → inline bold/italic → splice back.
//
// Placeholders use the SOH control character (\x01) as a delimiter, plus
// a base-26 (letter-only) index. Crucially they contain NO underscores or
// asterisks: an earlier placeholder format like "\x00PLACEHOLDER_0\x00"
// triggered the italic regex `_X_` to consume the underscore between
// "PLACEHOLDER" and the digit, splicing two adjacent placeholders into
// one mangled italic tag and leaving the text "PLACEHOLDER10" in the
// final output where the inline-code substitution should have been.
func markdownToTelegramHTML(s string) string {
	if s == "" {
		return s
	}

	type placeholder struct {
		token string
		html  string
	}
	var holds []placeholder

	stash := func(html string) string {
		token := "\x01MD" + indexToken(len(holds)) + "MD\x01"
		holds = append(holds, placeholder{token: token, html: html})
		return token
	}

	// 1. Fenced code blocks.
	s = reCodeBlock.ReplaceAllStringFunc(s, func(match string) string {
		groups := reCodeBlock.FindStringSubmatch(match)
		lang := strings.TrimSpace(groups[1])
		code := groups[2]
		var html string
		if lang == "" {
			html = "<pre>" + escapeHTML(code) + "</pre>"
		} else {
			html = `<pre><code class="language-` + escapeHTML(lang) + `">` + escapeHTML(code) + "</code></pre>"
		}
		return stash(html)
	})

	// 2. Inline code.
	s = reInlineCode.ReplaceAllStringFunc(s, func(match string) string {
		code := match[1 : len(match)-1]
		return stash("<code>" + escapeHTML(code) + "</code>")
	})

	// 3. Escape remaining HTML-sensitive characters.
	s = escapeHTML(s)

	// 4. Inline bold and italic. Operating on the already-escaped string is
	//    safe because both delimiters survive escapeHTML untouched.
	s = reBold.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[2 : len(match)-2]
		return "<b>" + inner + "</b>"
	})
	s = reItalic.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[1 : len(match)-1]
		return "<i>" + inner + "</i>"
	})

	// 5. Splice placeholders back. Done last so escapeHTML never touched
	//    the pre-built HTML inside fenced or inline code blocks.
	for _, p := range holds {
		s = strings.Replace(s, p.token, p.html, 1)
	}
	return s
}

// escapeHTML escapes the three characters Telegram's HTML parser cares
// about. Using a single pass instead of html.EscapeString avoids escaping
// quotes, which Telegram does not require and which leaks &#34; into prose.
func escapeHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// indexToken renders a placeholder index using lowercase letters only —
// no digits, no underscores, no asterisks — so neither the bold nor the
// italic regex can find a markdown delimiter inside a placeholder. Index
// 0 → "a", 26 → "ba", 27 → "bb", etc.
func indexToken(n int) string {
	if n < 0 {
		n = 0
	}
	if n == 0 {
		return "a"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte('a' + n%26)}, buf...)
		n /= 26
	}
	return string(buf)
}
