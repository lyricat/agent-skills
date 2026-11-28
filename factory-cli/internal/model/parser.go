package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Folder constants aligned with factory/scripts/board.py and factory/SKILL.md.
const (
	FolderBacklog = "backlog"
	FolderNew     = "new"
	FolderReview  = "review"
	FolderReady   = "ready"
	FolderDone    = "done"
	FolderStop    = "stop"
)

// MainFolders lists the standard rail stages shown by default.
var MainFolders = []string{FolderBacklog, FolderNew, FolderReview, FolderReady, FolderDone}

// AllFolders lists every board folder including stop/.
var AllFolders = []string{FolderBacklog, FolderNew, FolderReview, FolderReady, FolderDone, FolderStop}

// Turn identifies who acts next on an item.
type Turn string

const (
	TurnNone  Turn = ""
	TurnAgent Turn = "agent"
	TurnYou   Turn = "you"
)

var (
	itemFileRe  = regexp.MustCompile(`^(?:(?:([A-Z]+)(\d+))|(\d{4}))-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$|^([a-z0-9]+(?:-[a-z0-9]+)*)\.md$`)
	sectionRe   = regexp.MustCompile(`^## (.+?) · (?:(plan|question|review) · )?(\d{4}-\d{2}-\d{2} \d{2}:\d{2})\s*$`)
	verdictRe   = regexp.MustCompile(`(?m)^verdict:\s*(pass|fix)\b`)
	headSHARe   = regexp.MustCompile(`(?m)^head:\s*([0-9a-f]{7,})\s*$`)
	ansiCSIRe   = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	ansiOSCRe   = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	ansiOtherRe = regexp.MustCompile(`\x1b[@-Z\\-_]`)
)

// ItemIdentity captures parsed filename parts for an item file.
type ItemIdentity struct {
	Number *int
	Slug   string
	ID     string
	Label  string
	Bare   string
}

// Section represents a parsed "## who · [kind · ]YYYY-MM-DD HH:MM" section.
type Section struct {
	Who     string `json:"who"`
	Kind    string `json:"kind,omitempty"`
	Time    string `json:"time"`
	Body    string `json:"body"`
	Verdict string `json:"verdict,omitempty"`
	Head    string `json:"head,omitempty"`
}

// ParsedItem is the raw parsed contents of a Markdown item file.
type ParsedItem struct {
	Header   map[string]string
	Title    string
	Sections []Section
	Problems []string
}

// ParseFilename validates and extracts identity fields from an item filename.
// Note: unlike board.py's single regex, itemFileRe uses alternation to keep prefix separate while preserving exact semantics:
// <PREFIX><N>-<slug>.md, <NNNN>-<slug>.md, or <slug>.md (allowed only in backlog/ and new/).
func ParseFilename(folder, name, prefix string) (ItemIdentity, bool) {
	m := itemFileRe.FindStringSubmatch(name)
	if m == nil {
		return ItemIdentity{}, false
	}
	// Group 1: prefix letters, Group 2: digits with prefix
	// Group 3: 4 digits (legacy), Group 4: slug for numbered
	// Group 5: slug for unnumbered
	var numStr, slug string
	hasPrefix := false
	switch {
	case m[1] != "" && m[2] != "":
		hasPrefix = true
		numStr = m[2]
		slug = m[4]
	case m[3] != "":
		numStr = m[3]
		slug = m[4]
	case m[5] != "":
		slug = m[5]
	default:
		return ItemIdentity{}, false
	}

	if numStr == "" && folder != FolderBacklog && folder != FolderNew {
		return ItemIdentity{}, false
	}

	ident := ItemIdentity{
		Slug: slug,
		Bare: slug + ".md",
	}
	if numStr != "" {
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return ItemIdentity{}, false
		}
		ident.Number = &n
		ident.ID = strconv.Itoa(n)
		if hasPrefix {
			ident.Label = strings.SplitN(name, "-", 2)[0]
		} else {
			ident.Label = fmt.Sprintf("%s%d", prefix, n)
		}
	} else {
		ident.ID = slug
		ident.Label = "new"
	}
	return ident, true
}

// ParseItemContent parses frontmatter, title, thread sections, and format problems.
func ParseItemContent(text string, signers map[string]bool) ParsedItem {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	// Match Python str.splitlines() for trailing newline behavior.
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(normalized, "\n") {
		lines = lines[:len(lines)-1]
	}

	header := make(map[string]string)
	var problems []string
	i := 0

	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		closed := false
		for j := 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				for _, line := range lines[1:j] {
					k, v, ok := strings.Cut(line, ":")
					if ok {
						header[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
				}
				i = j + 1
				closed = true
				break
			}
		}
		if !closed {
			problems = append(problems, "the header has no closing ---")
		}
	}

	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	title := ""
	if i < len(lines) && strings.HasPrefix(lines[i], "# ") {
		title = strings.TrimSpace(lines[i][2:])
		i++
	} else {
		problems = append(problems, "no # title line")
	}

	var sections []Section
	var bodyLines []string
	hasCurrent := false

	flushSection := func() {
		if !hasCurrent {
			return
		}
		idx := len(sections) - 1
		body := strings.Trim(strings.Join(bodyLines, "\n"), "\n")
		sections[idx].Body = body
		if sections[idx].Kind == "review" {
			if vm := verdictRe.FindStringSubmatch(body); vm != nil {
				sections[idx].Verdict = vm[1]
			}
			if hm := headSHARe.FindStringSubmatch(body); hm != nil {
				sections[idx].Head = hm[1]
			}
		}
		bodyLines = nil
	}

	for _, line := range lines[i:] {
		if strings.HasPrefix(line, "## ") {
			if m := sectionRe.FindStringSubmatch(line); m != nil {
				flushSection()
				sections = append(sections, Section{
					Who:  m[1],
					Kind: m[2],
					Time: m[3],
				})
				hasCurrent = true
				continue
			}
			firstToken := strings.SplitN(line[3:], " ", 2)[0]
			firstToken = strings.Trim(firstToken, "·:,")
			if signers[firstToken] {
				flushSection()
				problems = append(problems, "a section heading not in the form ## name · YYYY-MM-DD HH:MM")
				sections = append(sections, Section{
					Who: strings.TrimSpace(line[3:]),
				})
				hasCurrent = true
				continue
			}
		}
		if hasCurrent {
			bodyLines = append(bodyLines, line)
		} else if strings.TrimSpace(line) != "" {
			problems = append(problems, "text before the first section")
		}
	}
	flushSection()

	if len(sections) == 0 {
		problems = append(problems, "no sections")
	}

	return ParsedItem{
		Header:   header,
		Title:    title,
		Sections: sections,
		Problems: dedupSorted(problems),
	}
}

// DetermineTurn decides whose turn it is to act on the item, matching board.py:turn().
func DetermineTurn(folder string, sections []Section) Turn {
	switch folder {
	case FolderBacklog, FolderDone, FolderStop:
		return TurnNone
	case FolderNew:
		return TurnAgent
	case FolderReview:
		var lastFactory *Section
		for idx := range sections {
			if sections[idx].Who == "factory" {
				lastFactory = &sections[idx]
			}
		}
		if lastFactory != nil && lastFactory.Kind == "review" && lastFactory.Verdict == "fix" {
			return TurnYou
		}
		return TurnAgent
	case FolderReady:
		return TurnYou
	default:
		return TurnNone
	}
}

// AskText returns the action prompt shown when turn == TurnYou.
func AskText(folder string, turn Turn) string {
	if turn != TurnYou {
		return ""
	}
	switch folder {
	case FolderReview:
		return "Decide: fix or take it"
	case FolderReady:
		return "Check, then ship"
	default:
		return ""
	}
}

// SanitizeLine strips terminal escape sequences and control chars for single-line rendering.
func SanitizeLine(s string) string {
	cleaned := stripANSI(s)
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		if unicode.IsControl(r) {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// SanitizeText strips terminal escape sequences and unsafe control chars while preserving newlines.
func SanitizeText(s string) string {
	cleaned := stripANSI(strings.ReplaceAll(s, "\r\n", "\n"))
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		switch {
		case r == '\n':
			b.WriteRune('\n')
		case r == '\t':
			b.WriteString("  ")
		case unicode.IsControl(r):
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "\n")
}

func stripANSI(s string) string {
	s = ansiOSCRe.ReplaceAllString(s, "")
	s = ansiCSIRe.ReplaceAllString(s, "")
	s = ansiOtherRe.ReplaceAllString(s, "")
	return s
}

func dedupSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	// Deterministic small slice sort.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
