package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"factory-cli/internal/gitboard"
	"factory-cli/internal/model"
)

// StatusOptions configures non-interactive status output.
type StatusOptions struct {
	Stage       string
	MineOnly    bool
	IncludeStop bool
	JSON        bool
	ActiveRuns  []gitboard.ActiveRun
}

// RenderStatus prints the board summary and item list in text or JSON format.
func RenderStatus(w io.Writer, board *model.Board, opts StatusOptions) error {
	if opts.JSON {
		items := filterStatusItems(board, opts)
		payload := struct {
			RepoName   string               `json:"repo_name"`
			GitHubURL  string               `json:"github_url,omitempty"`
			Ref        string               `json:"ref"`
			ShortRef   string               `json:"short_ref"`
			CommitSHA  string               `json:"commit_sha"`
			Yours      int                  `json:"yours"`
			Agents     int                  `json:"agents"`
			SentBack   int                  `json:"sent_back"`
			ActiveRuns []gitboard.ActiveRun `json:"active_runs,omitempty"`
			Stages     []model.StageSummary `json:"stages"`
			Items      []model.Item         `json:"items"`
		}{
			RepoName:   board.RepoName,
			GitHubURL:  board.GitHubURL,
			Ref:        board.Ref,
			ShortRef:   board.ShortRef,
			CommitSHA:  board.CommitSHA,
			Yours:      board.Yours,
			Agents:     board.Agents,
			SentBack:   board.SentBack,
			ActiveRuns: opts.ActiveRuns,
			Stages:     board.Stages,
			Items:      items,
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	shortSHA := board.CommitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s - factory board (%s @ %s)\n", model.SanitizeLine(board.RepoName), board.ShortRef, shortSHA)
	gates := "Nothing at your gates"
	if board.Yours > 0 {
		gates = fmt.Sprintf("%d at your gates", board.Yours)
	}
	fmt.Fprintf(&b, "%s | %d with agent", gates, board.Agents)
	for _, st := range board.Stages {
		if st.Folder == model.FolderDone {
			fmt.Fprintf(&b, " | %d done", st.Count)
			break
		}
	}
	if board.SentBack > 0 {
		fmt.Fprintf(&b, " | %d sent back", board.SentBack)
	}
	if len(opts.ActiveRuns) > 0 {
		names := make([]string, 0, len(opts.ActiveRuns))
		for _, r := range opts.ActiveRuns {
			names = append(names, r.Summary())
		}
		fmt.Fprintf(&b, " | running: %s", strings.Join(names, ", "))
	}
	b.WriteByte('\n')

	folders := model.MainFolders
	if opts.IncludeStop || opts.Stage == model.FolderStop {
		folders = model.AllFolders
	}
	if opts.Stage != "" {
		folders = []string{opts.Stage}
	}

	stageMap := make(map[string]model.StageSummary, len(board.Stages))
	for _, st := range board.Stages {
		stageMap[st.Folder] = st
	}

	labelWidth := 4
	for _, it := range board.Items {
		if len(it.Label) > labelWidth {
			labelWidth = len(it.Label)
		}
	}

	for _, folder := range folders {
		var group []model.Item
		for _, it := range board.Items {
			if it.Folder != folder {
				continue
			}
			if opts.MineOnly && it.Turn != model.TurnYou {
				continue
			}
			group = append(group, it)
		}
		st := stageMap[folder]
		gateMark := ""
		if st.AtYourGate > 0 {
			gateMark = " [!]"
		}
		note := ""
		if folder == model.FolderBacklog {
			note = "never built"
		} else if folder == model.FolderStop {
			note = "parked"
		} else if st.AverageStay != "" {
			note = "avg stay " + st.AverageStay
		}

		head := fmt.Sprintf("%s (%d)%s", strings.ToUpper(folder), len(group), gateMark)
		if note != "" {
			fmt.Fprintf(&b, "\n%-40s %s\n", head, note)
		} else {
			fmt.Fprintf(&b, "\n%s\n", head)
		}
		if len(group) == 0 {
			b.WriteString("    Nothing here\n")
			continue
		}
		for _, it := range group {
			mark := " "
			if it.Turn == model.TurnYou {
				mark = "*"
			}
			title := model.SanitizeLine(it.Title)
			if it.Round > 1 {
				title += fmt.Sprintf("  (round %d)", it.Round)
			}
			if it.Running != "" {
				title += fmt.Sprintf("  [%s]", runningVerb(it.Running))
			}
			fmt.Fprintf(&b, "  %s %-*s  %4s  %s\n", mark, labelWidth, it.Label, model.FormatAge(it.Since, board.FetchedAt), title)
			pad := strings.Repeat(" ", labelWidth+12)
			if it.Ask != "" {
				fmt.Fprintf(&b, "%s> %s\n", pad, it.Ask)
			}
			if len(it.Problems) > 0 {
				fmt.Fprintf(&b, "%s! Check the file: %s\n", pad, strings.Join(it.Problems, "; "))
			}
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func filterStatusItems(board *model.Board, opts StatusOptions) []model.Item {
	var out []model.Item
	for _, it := range board.Items {
		if opts.Stage != "" && it.Folder != opts.Stage {
			continue
		}
		if opts.Stage == "" && !opts.IncludeStop && it.Folder == model.FolderStop {
			continue
		}
		if opts.MineOnly && it.Turn != model.TurnYou {
			continue
		}
		out = append(out, it)
	}
	return out
}

// RenderItemDetail prints full metadata, stage trail, and sections for one item.
func RenderItemDetail(w io.Writer, board *model.Board, item *model.Item, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s · %s\n", item.Label, item.Folder)
	fmt.Fprintf(&b, "%s\n", model.SanitizeLine(item.Title))
	if item.Running != "" {
		fmt.Fprintf(&b, "Running: %s\n", runningVerb(item.Running))
	}
	if item.Ask != "" {
		fmt.Fprintf(&b, "Action: %s (waiting %s)\n", item.Ask, model.FormatAge(item.Since, board.FetchedAt))
	} else if item.Turn == model.TurnAgent {
		fmt.Fprintf(&b, "Turn: agent (since %s)\n", model.FormatAge(item.Since, board.FetchedAt))
	}
	if item.Branch != "" {
		fmt.Fprintf(&b, "Branch: %s\n", model.SanitizeLine(item.Branch))
	}
	if item.Round > 0 {
		fmt.Fprintf(&b, "Round: %d\n", item.Round)
	}
	if item.Source != "" {
		fmt.Fprintf(&b, "Source: %s\n", model.SanitizeLine(item.Source))
	}
	if item.Link != "" {
		fmt.Fprintf(&b, "Compare: %s\n", item.Link)
	}
	if len(item.Problems) > 0 {
		fmt.Fprintf(&b, "Format problems: %s\n", strings.Join(item.Problems, "; "))
	}
	if len(item.Steps) > 0 {
		var trail []string
		for i, st := range item.Steps {
			last := i+1 == len(item.Steps)
			end := board.FetchedAt
			if !last {
				end = item.Steps[i+1].At
			}
			if last && st.Folder == model.FolderDone {
				trail = append(trail, st.Folder)
			} else {
				trail = append(trail, fmt.Sprintf("%s (%s)", st.Folder, model.FormatSpan(end.Sub(st.At))))
			}
		}
		fmt.Fprintf(&b, "Trail: %s\n", strings.Join(trail, " → "))
	}

	for _, s := range item.Sections {
		b.WriteString("\n----------------------------------------\n")
		who := model.SanitizeLine(s.Who)
		if s.Kind != "" {
			who += " · " + s.Kind
		}
		if s.Time != "" {
			who += " · " + s.Time
		}
		fmt.Fprintf(&b, "%s\n\n%s\n", who, model.SanitizeText(s.Body))
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// RenderBranches lists discovered factory branch candidates.
func RenderBranches(w io.Writer, candidates []gitboard.BranchCandidate, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(candidates)
	}
	var b strings.Builder
	for i, c := range candidates {
		marker := " "
		if i == 0 {
			marker = "*"
		}
		sha := c.CommitSHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		kind := "local"
		if c.IsRemote {
			kind = "remote"
		}
		fmt.Fprintf(&b, "%s %-28s  %-6s  %s  (%s)\n", marker, c.ShortName, kind, sha, c.Ref)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func runningVerb(kind string) string {
	switch kind {
	case "build":
		return "building"
	case "review":
		return "reviewing"
	default:
		return kind
	}
}
