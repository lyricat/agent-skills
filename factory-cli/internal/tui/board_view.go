package tui

import (
	"fmt"
	"strings"
	"time"

	"factory-cli/internal/gitboard"
	"factory-cli/internal/model"
	"github.com/charmbracelet/lipgloss"
)

func (m AppModel) renderHeader(width int) string {
	if m.Board == nil {
		return styleHeaderBox.Width(width).Render(styleTitle.Render("factory-cli") + "  " + styleMuted.Render("Loading board..."))
	}

	shortSHA := m.Board.CommitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	var leftParts []string
	leftParts = append(leftParts, styleTitle.Render(model.SanitizeLine(m.Board.RepoName)))
	branchPart := fmt.Sprintf("%s @ %s", m.Board.ShortRef, shortSHA)
	if len(m.Candidates) > 1 {
		branchPart += fmt.Sprintf(" [%d/%d]", m.CandidateIdx+1, len(m.Candidates))
	}
	leftParts = append(leftParts, stylePlan.Render(branchPart))

	if len(m.ActiveRuns) > 0 {
		kinds := make([]string, 0, len(m.ActiveRuns))
		for _, r := range m.ActiveRuns {
			kinds = append(kinds, r.Summary())
		}
		leftParts = append(leftParts, styleRunning.Render("running: "+strings.Join(kinds, ", ")))
	}

	var rightParts []string
	if m.Refreshing {
		rightParts = append(rightParts, stylePlan.Render("syncing..."))
	} else if m.LastFetchErr != "" {
		rightParts = append(rightParts, styleStop.Render("offline: "+truncateRunes(m.LastFetchErr, 28)))
	} else if !m.LastSyncAt.IsZero() {
		syncNote := "synced " + m.LastSyncAt.Format("15:04:05")
		if m.Opts.Interval > 0 && !m.Opts.NoFetch {
			syncNote += fmt.Sprintf(" (every %s)", m.Opts.Interval)
		}
		rightParts = append(rightParts, styleMuted.Render(syncNote))
	}

	line1 := strings.Join(leftParts, "  ")
	if len(rightParts) > 0 {
		right := strings.Join(rightParts, " ")
		gap := width - lipgloss.Width(line1) - lipgloss.Width(right)
		if gap >= 2 {
			line1 += strings.Repeat(" ", gap) + right
		} else {
			line1 += "  " + right
		}
	}

	var gateText string
	if m.Board.Yours > 0 {
		gateText = styleGate.Render(fmt.Sprintf("%d at your gates", m.Board.Yours))
	} else {
		gateText = styleMuted.Render("0 at your gates")
	}

	doneCount := 0
	for _, st := range m.Board.Stages {
		if st.Folder == model.FolderDone {
			doneCount = st.Count
			break
		}
	}
	line2Parts := []string{
		gateText,
		styleMuted.Render(fmt.Sprintf("%d with agent", m.Board.Agents)),
		styleMuted.Render(fmt.Sprintf("%d done", doneCount)),
	}
	if m.Board.SentBack > 0 {
		line2Parts = append(line2Parts, styleMuted.Render(fmt.Sprintf("%d sent back", m.Board.SentBack)))
	}
	if m.MineOnly {
		line2Parts = append(line2Parts, styleGate.Render("[filter: mine]"))
	}
	if m.ShowStop {
		line2Parts = append(line2Parts, styleMuted.Render("[+stop]"))
	}
	if len(m.LastDiff) > 0 && time.Since(m.LastDiffAt) < 20*time.Second {
		line2Parts = append(line2Parts, styleOK.Render("updated: "+truncateRunes(strings.Join(m.LastDiff, ", "), 36)))
	}
	if m.LocalSyncNote != "" {
		switch m.LocalSyncState {
		case gitboard.LocalSyncFastFwd:
			if time.Since(m.LocalSyncAt) < 20*time.Second {
				line2Parts = append(line2Parts, styleOK.Render(m.LocalSyncNote))
			}
		case gitboard.LocalSyncDirty, gitboard.LocalSyncDiverged:
			line2Parts = append(line2Parts, styleGate.Render(m.LocalSyncNote))
		case gitboard.LocalSyncError:
			line2Parts = append(line2Parts, styleStop.Render(m.LocalSyncNote))
		}
	}
	line2 := strings.Join(line2Parts, styleMuted.Render(" · "))

	return styleHeaderBox.Width(width).Render(line1 + "\n" + line2)
}

func (m AppModel) renderStageTabs(width int) string {
	tabs := m.activeStageTabs()
	stageMap := make(map[string]model.StageSummary)
	totalMain := 0
	if m.Board != nil {
		for _, st := range m.Board.Stages {
			stageMap[st.Folder] = st
			if st.Folder != model.FolderStop {
				totalMain += st.Count
			}
		}
	}

	var rendered []string
	for idx, tab := range tabs {
		var label string
		hasGate := false
		if tab == "" {
			count := totalMain
			if m.ShowStop {
				count += stageMap[model.FolderStop].Count
			}
			label = fmt.Sprintf("all(%d)", count)
		} else {
			st := stageMap[tab]
			hasGate = st.AtYourGate > 0
			label = fmt.Sprintf("%s(%d)", tab, st.Count)
			if hasGate {
				label += "!"
			}
		}
		if idx == m.StageIdx {
			rendered = append(rendered, styleTabActive.Render(label))
		} else if hasGate {
			rendered = append(rendered, styleTabGate.Render(label))
		} else {
			rendered = append(rendered, styleTabInactive.Render(label))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(rendered, " "))
}

func (m AppModel) renderListView(width, bodyHeight int) string {
	if m.Board == nil {
		if m.InitialErr != "" {
			return styleStop.Render("Failed to load factory board: " + m.InitialErr)
		}
		return styleMuted.Render("Discovering factory branch...")
	}

	tabsLine := m.renderStageTabs(width)
	var lines []string
	lines = append(lines, tabsLine)

	if m.Mode == ModeSearch || strings.TrimSpace(m.SearchInput.Value()) != "" {
		searchLine := "Filter: " + m.SearchInput.View()
		if m.Mode != ModeSearch {
			searchLine = styleMuted.Render("Search: ") + stylePlan.Render(m.SearchInput.Value()) + styleMuted.Render(" (press / to edit, Esc to clear)")
		}
		lines = append(lines, searchLine)
	}

	lines = append(lines, styleDivider.Render(strings.Repeat("─", maxInt(10, width))))

	items := m.filteredItems()
	listRows := maxInt(1, bodyHeight-len(lines))

	if len(items) == 0 {
		lines = append(lines, styleMuted.Render("  No matching items."))
		for len(lines) < bodyHeight {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	// Calculate scroll window keeping SelectedIdx visible.
	start := 0
	if m.SelectedIdx >= listRows {
		start = m.SelectedIdx - listRows + 1
	}
	end := minInt(len(items), start+listRows)

	showStageCol := m.currentStageTab() == ""
	now := m.Board.FetchedAt
	if now.IsZero() {
		now = time.Now()
	}

	for i := start; i < end; i++ {
		it := items[i]
		selected := i == m.SelectedIdx

		cursor := "  "
		if selected {
			cursor = "› "
		}
		mark := " "
		if it.Turn == model.TurnYou {
			mark = "*"
		}

		stageTag := ""
		if showStageCol {
			stageTag = fmt.Sprintf("%-7s ", it.Folder)
		}
		age := fmt.Sprintf("%4s", model.FormatAge(it.Since, now))
		roundNote := ""
		if it.Round > 1 {
			roundNote = fmt.Sprintf(" (r%d)", it.Round)
		}
		runTag := ""
		if it.Running != "" {
			runTag = " [● " + runningVerb(it.Running) + "]"
		}
		cue := ""
		if it.Ask != "" {
			cue = " · " + it.Ask
		} else if len(it.Problems) > 0 {
			cue = " · !format"
		}

		prefixPlain := fmt.Sprintf("%s%s %-6s %s%s  ", cursor, mark, it.Label, stageTag, age)
		availTitle := maxInt(12, width-lipgloss.Width(prefixPlain)-lipgloss.Width(roundNote)-lipgloss.Width(runTag)-lipgloss.Width(cue))
		title := truncateRunes(model.SanitizeLine(it.Title), availTitle)

		if selected {
			row := prefixPlain + title + roundNote + runTag + cue
			pad := maxInt(0, width-lipgloss.Width(row))
			lines = append(lines, styleSelectedRow.Render(row+strings.Repeat(" ", pad)))
			continue
		}

		markStyled := mark
		if it.Turn == model.TurnYou {
			markStyled = styleGate.Render(mark)
		}
		labelStyled := stylePlan.Render(fmt.Sprintf("%-6s", it.Label))
		stageStyled := ""
		if showStageCol {
			stageStyled = styleMuted.Render(stageTag)
		}
		ageStyled := styleMuted.Render(age)
		runStyled := ""
		if runTag != "" {
			runStyled = styleRunning.Render(runTag)
		}
		cueStyled := ""
		if it.Ask != "" {
			cueStyled = styleGate.Render(cue)
		} else if len(it.Problems) > 0 {
			cueStyled = styleStop.Render(cue)
		}
		row := fmt.Sprintf("%s%s %s %s%s  %s%s%s%s",
			cursor, markStyled, labelStyled, stageStyled, ageStyled, title, styleMuted.Render(roundNote), runStyled, cueStyled)
		lines = append(lines, row)
	}

	for len(lines) < bodyHeight {
		lines = append(lines, "")
	}
	return strings.Join(lines[:bodyHeight], "\n")
}

func (m AppModel) renderHelpView(width, bodyHeight int) string {
	helpLines := []string{
		styleTitle.Render("Keybindings"),
		"",
		"  j / k / ↑ / ↓      Move cursor / scroll detail",
		"  h / l / Tab        Switch stage tab",
		"  1 - 6              Jump directly to stage tab",
		"  Enter              Open item detail",
		"  Esc / Backspace    Close detail or clear search",
		"  n / p              Next / previous section in detail view",
		"  /                  Filter items by ID, title, or branch",
		"  m                  Toggle 'at your gates' filter",
		"  s                  Toggle stop/ stage visibility",
		"  b                  Switch discovered factory branch (if multiple)",
		"  r                  Fetch and refresh immediately",
		"  ?                  Toggle this help",
		"  q / Ctrl+C         Quit",
	}
	for len(helpLines) < bodyHeight {
		helpLines = append(helpLines, "")
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(helpLines[:bodyHeight], "\n"))
}

func (m AppModel) renderFooter(width int) string {
	var hints string
	switch m.Mode {
	case ModeDetail:
		hints = fmt.Sprintf("j/k scroll  n/p section  r refresh  Esc back  q quit  (%3.0f%%)", m.DetailViewport.ScrollPercent()*100)
	case ModeSearch:
		hints = "Type to filter  Enter apply  Esc cancel"
	case ModeHelp:
		hints = "Press Esc or ? to close help"
	default:
		hints = "j/k move  h/l stage  Enter detail  / search  m mine  s stop  b branch  r refresh  ? help  q quit"
	}
	return styleFooterBox.Width(width).Render(truncateRunes(hints, width))
}

func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
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
