package gitboard

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"factory-cli/internal/model"
)

const lockLeaseWindow = 10 * time.Minute

var (
	buildItemDirRe  = regexp.MustCompile(`^([A-Z]+\d+|\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*$`)
	reviewItemDirRe = regexp.MustCompile(`^review-([A-Z]+\d+|\d{4})$`)
)

// ActiveRun represents a currently active factory run detected via its lock directory.
type ActiveRun struct {
	Kind  string    `json:"kind"`
	Since time.Time `json:"since"`
	Items []string  `json:"items,omitempty"`
}

// Summary formats a compact label such as "build (FT16)" or "ship".
func (r ActiveRun) Summary() string {
	if len(r.Items) == 0 {
		return r.Kind
	}
	labels := make([]string, 0, len(r.Items))
	for _, item := range r.Items {
		if id, _, ok := strings.Cut(item, "-"); ok && id != "" {
			labels = append(labels, id)
		} else {
			labels = append(labels, item)
		}
	}
	return r.Kind + " (" + strings.Join(labels, ",") + ")"
}

// DetectActiveRuns checks <root>/<OWNER>/<REPO>/{build,review,ship}/lock for recent locks
// and identifies active per-item build/review directories.
func DetectActiveRuns(githubSlug, rootOverride string, now time.Time) []ActiveRun {
	if strings.TrimSpace(githubSlug) == "" {
		return nil
	}
	parts := strings.Split(githubSlug, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}

	root := strings.TrimSpace(rootOverride)
	if root == "" {
		root = strings.TrimSpace(os.Getenv("FACTORY_ROOT"))
	}
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return nil
		}
		root = filepath.Join(home, ".factory")
	}

	var active []ActiveRun
	for _, kind := range []string{"build", "review", "ship"} {
		runDir := filepath.Join(root, parts[0], parts[1], kind)
		lockPath := filepath.Join(runDir, "lock")
		info, err := os.Stat(lockPath)
		if err != nil {
			continue
		}
		if !isFreshMod(info.ModTime(), now) {
			continue
		}
		var items []string
		if kind == "build" || kind == "review" {
			items = detectRunItems(runDir, kind, now)
		}
		active = append(active, ActiveRun{
			Kind:  kind,
			Since: info.ModTime(),
			Items: items,
		})
	}
	return active
}

func isFreshMod(modTime, now time.Time) bool {
	age := now.Sub(modTime)
	return age >= -time.Minute && age < lockLeaseWindow
}

type detectedEntry struct {
	key     string
	modTime time.Time
}

func detectRunItems(runDir, kind string, now time.Time) []string {
	tmpDir := filepath.Join(runDir, "tmp")
	if entries, err := os.ReadDir(tmpDir); err == nil {
		var matched []detectedEntry
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			key, ok := extractRunItemKey(kind, ent.Name())
			if !ok {
				continue
			}
			p := filepath.Join(tmpDir, ent.Name())
			if t, recent := recentDirActivity(p, now, true); recent {
				matched = append(matched, detectedEntry{key: key, modTime: t})
			}
		}
		if len(matched) > 0 {
			return sortAndDedupEntries(matched)
		}
	}

	entries, err := os.ReadDir(runDir)
	if err != nil {
		return nil
	}
	var allCandidates []string
	var recentMatched []detectedEntry
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		key, ok := extractRunItemKey(kind, ent.Name())
		if !ok {
			continue
		}
		allCandidates = append(allCandidates, key)
		p := filepath.Join(runDir, ent.Name())
		if t, recent := recentWorktreeActivity(p, now); recent {
			recentMatched = append(recentMatched, detectedEntry{key: key, modTime: t})
		}
	}
	if len(recentMatched) > 0 {
		return sortAndDedupEntries(recentMatched)
	}
	if kind == "review" && len(allCandidates) == 1 {
		return allCandidates
	}
	return nil
}

func extractRunItemKey(kind, dirName string) (string, bool) {
	switch kind {
	case "build":
		if buildItemDirRe.MatchString(dirName) {
			return dirName, true
		}
	case "review":
		if m := reviewItemDirRe.FindStringSubmatch(dirName); m != nil {
			return m[1], true
		}
	}
	return "", false
}

func recentDirActivity(dirPath string, now time.Time, checkFactorySubdir bool) (time.Time, bool) {
	var latest time.Time
	consider := func(p string) {
		info, err := os.Stat(p)
		if err != nil {
			return
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	consider(dirPath)
	if checkFactorySubdir {
		consider(filepath.Join(dirPath, "factory"))
		if entries, err := os.ReadDir(filepath.Join(dirPath, "factory")); err == nil {
			for _, ent := range entries {
				if info, err := ent.Info(); err == nil && info.ModTime().After(latest) {
					latest = info.ModTime()
				}
			}
		}
	}
	if latest.IsZero() {
		return time.Time{}, false
	}
	return latest, isFreshMod(latest, now)
}

// recentWorktreeActivity checks bounded metadata paths in a git worktree without deep-walking build artifacts.
func recentWorktreeActivity(wtPath string, now time.Time) (time.Time, bool) {
	var latest time.Time
	consider := func(p string) {
		info, err := os.Stat(p)
		if err != nil {
			return
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}

	consider(wtPath)
	dotGitPath := filepath.Join(wtPath, ".git")
	consider(dotGitPath)
	if data, err := os.ReadFile(dotGitPath); err == nil {
		if gitDirLine, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:"); ok {
			gd := strings.TrimSpace(gitDirLine)
			if !filepath.IsAbs(gd) {
				gd = filepath.Join(wtPath, gd)
			}
			consider(filepath.Join(gd, "index"))
			consider(filepath.Join(gd, "HEAD"))
		}
	}
	if entries, err := os.ReadDir(wtPath); err == nil {
		for _, ent := range entries {
			if info, err := ent.Info(); err == nil && info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
	}
	if latest.IsZero() {
		return time.Time{}, false
	}
	return latest, isFreshMod(latest, now)
}

func sortAndDedupEntries(entries []detectedEntry) []string {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].modTime.After(entries[j].modTime)
	})
	seen := make(map[string]bool, len(entries))
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !seen[e.key] {
			seen[e.key] = true
			out = append(out, e.key)
		}
	}
	return out
}

// ApplyActiveRuns returns a shallow copy of board with Item.Running populated from active runs.
func ApplyActiveRuns(board *model.Board, runs []ActiveRun) *model.Board {
	if board == nil {
		return nil
	}
	cloned := *board
	cloned.Items = make([]model.Item, len(board.Items))
	copy(cloned.Items, board.Items)

	prefix := board.Config.Prefix
	for i := range cloned.Items {
		cloned.Items[i].Running = matchItemRunningKind(cloned.Items[i], prefix, runs)
	}
	return &cloned
}

func matchItemRunningKind(item model.Item, prefix string, runs []ActiveRun) string {
	var matched []string
	for _, r := range runs {
		if r.Kind != "build" && r.Kind != "review" {
			continue
		}
		for _, entry := range r.Items {
			if runEntryMatchesItem(entry, item, prefix) {
				matched = append(matched, r.Kind)
				break
			}
		}
	}
	if len(matched) == 0 {
		return ""
	}
	if len(matched) == 1 {
		return matched[0]
	}
	for _, k := range matched {
		if (item.Folder == model.FolderReview && k == "review") ||
			((item.Folder == model.FolderNew || item.Folder == model.FolderBacklog) && k == "build") {
			return k
		}
	}
	return matched[0]
}

func runEntryMatchesItem(entry string, item model.Item, prefix string) bool {
	stem := strings.TrimSuffix(item.Name, ".md")
	if strings.EqualFold(entry, stem) || strings.EqualFold(entry, item.Label) || entry == item.ID {
		return true
	}
	token, _, _ := strings.Cut(entry, "-")
	if token == "" {
		return false
	}
	if strings.EqualFold(token, item.Label) || token == item.ID {
		return true
	}
	if len(token) == 4 && isAllDigits(token) && prefix != "" {
		if n, err := strconv.Atoi(token); err == nil {
			normalized := prefix + strconv.Itoa(n)
			if strings.EqualFold(normalized, item.Label) {
				return true
			}
		}
	}
	return false
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}
