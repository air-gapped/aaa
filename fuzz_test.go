package main

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// FuzzParse feeds arbitrary argument lists (NUL-separated) to the parser.
func FuzzParse(f *testing.F) {
	for _, seed := range []string{"", "fix the thing", "--close\x00k7f", "--close\x00--all", "--edit\x00k7f\x00new text",
		"--show\x00k7f\x00--json", "--clsoe\x00k7f", "-", "--", "\x00\x00", "--json\x00--skill", strings.Repeat("ä", 1001)} {
		f.Add(seed)
	}
	known := map[string]bool{"": true, "--close": true, "--edit": true, "--show": true, "--skill": true, "--help": true, "--version": true}
	f.Fuzz(func(t *testing.T, joined string) {
		args := strings.Split(joined, "\x00")
		c, err := parse(args)
		if err != nil {
			return
		}
		if !known[c.op] {
			t.Fatalf("unknown op %q from %q", c.op, args)
		}
		for _, w := range c.words {
			if len(w) > 1 && w[0] == '-' {
				t.Fatalf("flag-like word %q accepted as text from %q", w, args)
			}
		}
		if n := utf8.RuneCountInString(c.text()); n > maxTextRunes {
			t.Fatalf("accepted %d-character text", n)
		}
		if c.op == "" && len(c.words) > 0 && strings.TrimSpace(c.text()) == "" {
			t.Fatalf("accepted blank capture text from %q", args)
		}
	})
}

// FuzzClean checks that nothing printed through clean can drive a terminal.
func FuzzClean(f *testing.F) {
	for _, seed := range []string{"plain", "\x1b]52;c;x\x07", "\u009b", "a\x9bb", "�", "\xff\xfe", "tab\t", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := clean(s)
		if !utf8.ValidString(out) {
			t.Fatalf("clean(%q) = %q is not valid UTF-8", s, out)
		}
		if strings.ContainsFunc(out, unicode.IsControl) {
			t.Fatalf("clean(%q) = %q still contains a control character", s, out)
		}
		if clean(out) != out {
			t.Fatalf("clean is not idempotent on %q", s)
		}
		if utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl) && out != s {
			t.Fatalf("clean changed safe text %q to %q", s, out)
		}
	})
}
