package model

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Move records when an item entered a board folder.
type Move struct {
	Folder string    `json:"folder"`
	At     time.Time `json:"at"`
}

// Item represents a parsed factory item and its git history.
type Item struct {
	Number   *int      `json:"number,omitempty"`
	Slug     string    `json:"slug"`
	Folder   string    `json:"folder"`
	File     string    `json:"file"`
	Name     string    `json:"name"`
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	Title    string    `json:"title"`
	Source   string    `json:"source,omitempty"`
	Branch   string    `json:"branch,omitempty"`
	Round    int       `json:"round,omitempty"`
	RawRound string    `json:"raw_round,omitempty"`
	Sections []Section `json:"sections"`
	Problems []string  `json:"problems,omitempty"`
	Steps    []Move    `json:"steps,omitempty"`
	Changed  time.Time `json:"changed,omitempty"`
	Since    time.Time `json:"since,omitempty"`
	Turn     Turn      `json:"turn"`
	Ask      string    `json:"ask,omitempty"`
	Link     string    `json:"link,omitempty"`
	Running  string    `json:"running,omitempty"`
}

// StageSummary captures counts and average stay for one stage folder.
type StageSummary struct {
	Folder      string `json:"folder"`
	Count       int    `json:"count"`
	AtYourGate  int    `json:"at_your_gate"`
	AverageStay string `json:"average_stay,omitempty"`
}

// Board is a point-in-time snapshot of a repository's factory_board branch.
type Board struct {
	RepoName   string         `json:"repo_name"`
	GitHubURL  string         `json:"github_url,omitempty"`
	GitHubSlug string         `json:"github_slug,omitempty"`
	Ref        string         `json:"ref"`
	ShortRef   string         `json:"short_ref"`
	CommitSHA  string         `json:"commit_sha"`
	Config     Config         `json:"config"`
	Location   *time.Location `json:"-"`
	FetchedAt  time.Time      `json:"fetched_at"`
	Items      []Item         `json:"items"`
	Stages     []StageSummary `json:"stages"`
	Yours      int            `json:"yours"`
	Agents     int            `json:"agents"`
	SentBack   int            `json:"sent_back"`
}

// ComputeWaitingSince matches board.py:waiting_since().
func ComputeWaitingSince(item Item, loc *time.Location) time.Time {
	if item.Turn != TurnYou {
		return item.Changed
	}
	if loc == nil {
		loc = time.Local
	}
	latest := item.Changed
	for _, s := range item.Sections {
		if s.Time == "" {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02 15:04", s.Time, loc)
		if err != nil {
			continue
		}
		if t.After(latest) {
			latest = t
		}
	}
	return latest
}

// AverageStay computes mean completed stay duration in folder, matching board.py:average_stay().
func AverageStay(items []Item, folder string) time.Duration {
	var total time.Duration
	var count int
	for _, it := range items {
		for i := 0; i+1 < len(it.Steps); i++ {
			if it.Steps[i].Folder == folder {
				d := it.Steps[i+1].At.Sub(it.Steps[i].At)
				if d < 0 {
					d = 0
				}
				total += d
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}

// BuildBoard normalizes items, computes summary metrics, and orders items per stage.
func BuildBoard(repoName, githubURL, githubSlug, ref, shortRef, commitSHA string, cfg Config, loc *time.Location, items []Item, now time.Time) *Board {
	if loc == nil {
		loc = time.Local
	}
	byFolder := make(map[string][]Item, len(AllFolders))
	for i := range items {
		it := items[i]
		if it.Title == "" {
			it.Title = strings.ReplaceAll(it.Slug, "-", " ")
		}
		if r, err := strconv.Atoi(it.RawRound); err == nil && r > 0 {
			it.Round = r
		}
		it.Turn = DetermineTurn(it.Folder, it.Sections)
		it.Ask = AskText(it.Folder, it.Turn)
		it.Since = ComputeWaitingSince(it, loc)
		byFolder[it.Folder] = append(byFolder[it.Folder], it)
	}

	ordered := make([]Item, 0, len(items))
	stages := make([]StageSummary, 0, len(AllFolders))
	var yours, agents, sentBack int

	for _, folder := range AllFolders {
		group := byFolder[folder]
		SortStageItems(folder, group)

		var gateCount int
		for _, it := range group {
			ordered = append(ordered, it)
			if folder != FolderStop {
				if it.Turn == TurnYou {
					yours++
					gateCount++
				} else if it.Turn == TurnAgent {
					agents++
				}
				if it.Round > 1 {
					sentBack += it.Round - 1
				}
			}
		}

		avg := ""
		if folder == FolderNew || folder == FolderReview || folder == FolderReady {
			if d := AverageStay(ordered, folder); d > 0 {
				avg = FormatSpan(d)
			}
		}
		stages = append(stages, StageSummary{
			Folder:      folder,
			Count:       len(group),
			AtYourGate:  gateCount,
			AverageStay: avg,
		})
	}

	// Recompute average stays against all items (not only prior folders).
	for i := range stages {
		f := stages[i].Folder
		if f == FolderNew || f == FolderReview || f == FolderReady {
			if d := AverageStay(ordered, f); d > 0 {
				stages[i].AverageStay = FormatSpan(d)
			} else {
				stages[i].AverageStay = ""
			}
		}
	}

	return &Board{
		RepoName:   repoName,
		GitHubURL:  githubURL,
		GitHubSlug: githubSlug,
		Ref:        ref,
		ShortRef:   shortRef,
		CommitSHA:  commitSHA,
		Config:     cfg,
		Location:   loc,
		FetchedAt:  now,
		Items:      ordered,
		Stages:     stages,
		Yours:      yours,
		Agents:     agents,
		SentBack:   sentBack,
	}
}

// SortStageItems sorts done descending by Changed, and all other stages ascending by Since.
func SortStageItems(folder string, items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if folder == FolderDone {
			if !items[i].Changed.Equal(items[j].Changed) {
				return items[i].Changed.After(items[j].Changed)
			}
		} else {
			if !items[i].Since.Equal(items[j].Since) {
				return items[i].Since.Before(items[j].Since)
			}
		}
		return items[i].File < items[j].File
	})
}

// ItemsInStage returns items belonging to the given folder (or all main folders if folder == "").
func (b *Board) ItemsInStage(folder string, includeStop bool) []Item {
	if b == nil {
		return nil
	}
	var out []Item
	for _, it := range b.Items {
		if folder == "" {
			if it.Folder == FolderStop && !includeStop {
				continue
			}
			out = append(out, it)
		} else if it.Folder == folder {
			out = append(out, it)
		}
	}
	return out
}

// FormatAge formats elapsed time since t in whole units (1m, 4h, 3d), matching board.py:age().
func FormatAge(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	secs := int(now.Sub(t).Seconds())
	if secs < 0 {
		secs = 0
	}
	if secs < 3600 {
		m := secs / 60
		if m < 1 {
			m = 1
		}
		return fmt.Sprintf("%dm", m)
	}
	if secs < 86400 {
		return fmt.Sprintf("%dh", secs/3600)
	}
	return fmt.Sprintf("%dd", secs/86400)
}

// FormatSpan formats a duration into compact comparative form (40m, 19h, 1.2d), matching board.py:span().
func FormatSpan(d time.Duration) string {
	secs := d.Seconds()
	if secs < 0 {
		secs = 0
	}
	if secs < 3600 {
		m := int(secs / 60)
		if m < 1 {
			m = 1
		}
		return fmt.Sprintf("%dm", m)
	}
	if secs < 86400 {
		return fmt.Sprintf("%dh", int(math.Round(secs/3600)))
	}
	s := fmt.Sprintf("%.1fd", secs/86400)
	return strings.ReplaceAll(s, ".0d", "d")
}

// FindItem resolves an item by label (FT16), numeric ID (16), slug, or unambiguous prefix.
func FindItem(board *Board, query string) (*Item, error) {
	if board == nil {
		return nil, fmt.Errorf("board is empty")
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("empty item query")
	}
	ql := strings.ToLower(q)

	for i := range board.Items {
		it := &board.Items[i]
		if strings.EqualFold(it.Label, q) || strings.EqualFold(it.ID, q) || strings.EqualFold(it.Slug, q) {
			return it, nil
		}
	}

	var matches []*Item
	for i := range board.Items {
		it := &board.Items[i]
		if strings.HasPrefix(strings.ToLower(it.Label), ql) ||
			strings.HasPrefix(strings.ToLower(it.Slug), ql) ||
			strings.HasPrefix(strings.ToLower(it.Name), ql) {
			matches = append(matches, it)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		labels := make([]string, 0, len(matches))
		for _, m := range matches {
			labels = append(labels, m.Label)
		}
		return nil, fmt.Errorf("ambiguous item %q (matches: %s)", query, strings.Join(labels, ", "))
	}
	return nil, fmt.Errorf("item %q not found", query)
}

// DiffBoards summarizes changes between two snapshots for status notifications.
func DiffBoards(prev, next *Board) []string {
	if prev == nil || next == nil || prev.CommitSHA == next.CommitSHA {
		return nil
	}
	keyFor := func(it Item) string {
		if it.Label != "" && it.Label != "new" {
			return it.Label + ":" + it.Slug
		}
		return it.Name
	}
	displayName := func(it Item) string {
		if it.Label != "" && it.Label != "new" {
			return it.Label
		}
		return it.Slug
	}

	prevMap := make(map[string]Item, len(prev.Items))
	for _, it := range prev.Items {
		prevMap[keyFor(it)] = it
	}
	nextMap := make(map[string]Item, len(next.Items))
	for _, it := range next.Items {
		nextMap[keyFor(it)] = it
	}

	var diffs []string
	for k, nxt := range nextMap {
		prv, ok := prevMap[k]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("+%s %s", displayName(nxt), nxt.Folder))
			continue
		}
		if prv.Folder != nxt.Folder {
			diffs = append(diffs, fmt.Sprintf("%s %s → %s", displayName(nxt), prv.Folder, nxt.Folder))
		} else if len(prv.Sections) != len(nxt.Sections) || prv.Round != nxt.Round {
			diffs = append(diffs, fmt.Sprintf("%s updated", displayName(nxt)))
		}
	}
	for k, prv := range prevMap {
		if _, ok := nextMap[k]; !ok {
			diffs = append(diffs, fmt.Sprintf("-%s %s", displayName(prv), prv.Folder))
		}
	}
	sort.Strings(diffs)
	return diffs
}
