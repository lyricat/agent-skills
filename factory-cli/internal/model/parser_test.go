package model

import (
	"testing"
)

func TestParseFilename(t *testing.T) {
	cases := []struct {
		folder string
		name   string
		prefix string
		ok     bool
		label  string
		id     string
		slug   string
		bare   string
	}{
		{"new", "FT16-search-box.md", "FT", true, "FT16", "16", "search-box", "search-box.md"},
		{"review", "AB9-fix-login.md", "FT", true, "AB9", "9", "fix-login", "fix-login.md"},
		{"ready", "0012-legacy-task.md", "FT", true, "FT12", "12", "legacy-task", "legacy-task.md"},
		{"backlog", "rough-idea.md", "FT", true, "new", "rough-idea", "rough-idea", "rough-idea.md"},
		{"new", "fresh-feature.md", "FT", true, "new", "fresh-feature", "fresh-feature", "fresh-feature.md"},
		{"review", "unnumbered-not-allowed.md", "FT", false, "", "", "", ""},
		{"done", "unnumbered-not-allowed.md", "FT", false, "", "", "", ""},
		{"new", "README.txt", "FT", false, "", "", "", ""},
		{"new", "bad_slug.md", "FT", false, "", "", "", ""},
		{"new", ".gitkeep", "FT", false, "", "", "", ""},
	}

	for _, tc := range cases {
		ident, ok := ParseFilename(tc.folder, tc.name, tc.prefix)
		if ok != tc.ok {
			t.Fatalf("ParseFilename(%q, %q) ok=%v, want %v", tc.folder, tc.name, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if ident.Label != tc.label || ident.ID != tc.id || ident.Slug != tc.slug || ident.Bare != tc.bare {
			t.Fatalf("ParseFilename(%q, %q) got %+v, want label=%q id=%q slug=%q bare=%q",
				tc.folder, tc.name, ident, tc.label, tc.id, tc.slug, tc.bare)
		}
	}
}

func TestParseItemContentAndProblems(t *testing.T) {
	signers := map[string]bool{"factory": true, "alex": true}

	t.Run("valid item with review", func(t *testing.T) {
		raw := `---
source: issue #12
branch: factory/FT16-search-box
round: 2
---
# Search box in toolbar

## alex · 2026-11-04 09:15
Add filtering.

### Subheading inside body
Still alex's section.

## factory · review · 2026-11-05 10:00
head: 3f2a1c9e0b4d
verdict: fix
fix:
- [P1] src/sync.go:88: drop token from log
`
		parsed := ParseItemContent(raw, signers)
		if len(parsed.Problems) != 0 {
			t.Fatalf("unexpected problems: %v", parsed.Problems)
		}
		if parsed.Title != "Search box in toolbar" {
			t.Fatalf("title=%q", parsed.Title)
		}
		if parsed.Header["branch"] != "factory/FT16-search-box" || parsed.Header["round"] != "2" {
			t.Fatalf("unexpected header: %+v", parsed.Header)
		}
		if len(parsed.Sections) != 2 {
			t.Fatalf("len(sections)=%d, want 2", len(parsed.Sections))
		}
		if parsed.Sections[1].Kind != "review" || parsed.Sections[1].Verdict != "fix" || parsed.Sections[1].Head != "3f2a1c9e0b4d" {
			t.Fatalf("unexpected review section: %+v", parsed.Sections[1])
		}
		if turn := DetermineTurn(FolderReview, parsed.Sections); turn != TurnYou {
			t.Fatalf("DetermineTurn=%q, want %q", turn, TurnYou)
		}
	})

	t.Run("mistyped signer heading and unclosed header", func(t *testing.T) {
		raw := `---
branch: factory/FT1-bad
# Missing closing dash

text before section

## alex: forgot middle dot
hello
`
		parsed := ParseItemContent(raw, signers)
		if len(parsed.Problems) < 3 {
			t.Fatalf("expected multiple problems, got %v", parsed.Problems)
		}
	})
}

func TestSanitize(t *testing.T) {
	dirty := "Fix \x1b[31mred\x1b[0m\x1b]0;evil-title\x07\r\nbell\x07"
	line := SanitizeLine(dirty)
	if line != "Fix red  bell" {
		t.Fatalf("SanitizeLine=%q", line)
	}
	multi := SanitizeText("line1\x1b[2J\n\tline2\x00")
	if multi != "line1\n  line2 " {
		t.Fatalf("SanitizeText=%q", multi)
	}
}
