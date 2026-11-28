package tui

import (
	"fmt"
	"regexp"
	"strings"

	"factory-cli/internal/model"
	"github.com/charmbracelet/lipgloss"
)

var (
	sevLineRe   = regexp.MustCompile(`^\s*(?:[-*]\s*)?\[(P[0-3])\]\s*(.*)$`)
	verdictRe   = regexp.MustCompile(`^verdict:\s*(\w+)`)
	headLineRe  = regexp.MustCompile(`^head:\s*([0-9a-fA-F]{7,})`)
	sectionList = regexp.MustCompile(`^(fix|notes):\s*$`)
)

// renderDetailHeader renders the pinned task summary block displayed above the scrollable thread.
func renderDetailHeader(board *model.Board, item *model.Item, width int) string {
	if item == nil {
		return styleHeaderBox.Width(maxInt(40, width)).Render("No item selected.")
	}
	if width < 40 {
		width = 80
	}
	wrapWidth := width - 2

	var lines []string
	addLine := func(s string) {
		lines = append(lines, s)
	}
	addWrapped := func(prefix, text string, style lipgloss.Style) {
		rendered := style.Width(wrapWidth).Render(prefix + text)
		lines = append(lines, strings.Split(rendered, "\n")...)
	}

	badge := fmt.Sprintf("[%s · %s]", item.Label, item.Folder)
	addWrapped("", stylePlan.Render(badge)+" "+styleTitle.Render(model.SanitizeLine(item.Title)), lipgloss.NewStyle())

	if item.Running != "" {
		addLine(styleRunning.Render("● Running: " + runningVerb(item.Running)))
	}
	if item.Ask != "" {
		addLine(styleGate.Render(fmt.Sprintf("▶ %s (waiting %s)", item.Ask, model.FormatAge(item.Since, board.FetchedAt))))
	} else if item.Turn == model.TurnAgent {
		addLine(styleMuted.Render(fmt.Sprintf("● Waiting on agent (since %s)", model.FormatAge(item.Since, board.FetchedAt))))
	}
	if len(item.Problems) > 0 {
		addWrapped("! Format problem: ", strings.Join(item.Problems, "; "), styleStop)
	}

	if len(item.Steps) > 0 {
		var parts []string
		for i, st := range item.Steps {
			last := i+1 == len(item.Steps)
			end := board.FetchedAt
			if !last {
				end = item.Steps[i+1].At
			}
			if last && st.Folder == model.FolderDone {
				parts = append(parts, styleOK.Render(st.Folder))
			} else {
				span := model.FormatSpan(end.Sub(st.At))
				if last {
					parts = append(parts, styleTitle.Render(st.Folder)+" "+styleMuted.Render("("+span+")"))
				} else {
					parts = append(parts, st.Folder+" "+styleMuted.Render("("+span+")"))
				}
			}
		}
		addWrapped("", styleMuted.Render("Trail: ")+strings.Join(parts, styleMuted.Render(" → ")), lipgloss.NewStyle())
	}

	var facts []string
	if item.Branch != "" {
		facts = append(facts, "branch: "+model.SanitizeLine(item.Branch))
	}
	if item.Round > 0 {
		facts = append(facts, fmt.Sprintf("round: %d", item.Round))
	}
	if item.Source != "" {
		facts = append(facts, "source: "+model.SanitizeLine(item.Source))
	}
	if len(facts) > 0 {
		addWrapped("", strings.Join(facts, " · "), styleMuted)
	}

	return styleHeaderBox.Width(width).Render(strings.Join(lines, "\n"))
}

// renderDetailThread renders the scrollable thread sections (and optional compare link) below the pinned task header.
func renderDetailThread(board *model.Board, item *model.Item, width int) (string, []int) {
	if item == nil {
		return "No item selected.", nil
	}
	if width < 40 {
		width = 80
	}
	wrapWidth := width - 2

	var lines []string
	addLine := func(s string) {
		lines = append(lines, s)
	}
	addWrapped := func(prefix, text string, style lipgloss.Style) {
		rendered := style.Width(wrapWidth).Render(prefix + text)
		lines = append(lines, strings.Split(rendered, "\n")...)
	}

	if item.Link != "" {
		addWrapped("", "compare: "+item.Link, styleMuted)
	}

	// Thread sections.
	owners := board.Config.OwnerSet()
	sectionOffsets := make([]int, 0, len(item.Sections))
	rule := styleDivider.Render(strings.Repeat("─", minInt(wrapWidth, 72)))

	for idx, sec := range item.Sections {
		if len(lines) > 0 {
			addLine("")
		}
		sectionOffsets = append(sectionOffsets, len(lines))
		addLine(rule)

		who := model.SanitizeLine(sec.Who)
		whoStyle := styleMuted
		switch {
		case sec.Who == "factory":
			whoStyle = stylePlan
		case owners[sec.Who]:
			whoStyle = styleTitle
		}

		headerParts := []string{whoStyle.Render(who)}
		if sec.Kind != "" {
			kindStyle := styleMuted
			switch sec.Kind {
			case "plan":
				kindStyle = stylePlan
			case "question":
				kindStyle = styleGate
			case "review":
				switch sec.Verdict {
				case "pass":
					kindStyle = styleOK
				case "fix":
					kindStyle = styleStop
				default:
					kindStyle = stylePlan
				}
			}
			headerParts = append(headerParts, kindStyle.Render("["+sec.Kind+"]"))
		}
		if sec.Time != "" {
			headerParts = append(headerParts, styleMuted.Render(sec.Time))
		}
		addLine(fmt.Sprintf("%s  %s", strings.Join(headerParts, " · "), styleMuted.Render(fmt.Sprintf("#%d/%d", idx+1, len(item.Sections)))))
		addLine("")

		body := model.SanitizeText(sec.Body)
		if body == "" {
			addLine(styleMuted.Render("(empty)"))
			continue
		}
		inFence := false
		for _, rawLine := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(rawLine)
			if strings.HasPrefix(trimmed, "```") {
				inFence = !inFence
				addLine(styleMuted.Render(rawLine))
				continue
			}
			if inFence {
				addLine(styleCodeBlock.Render(rawLine))
				continue
			}
			if sec.Kind == "review" {
				if vm := verdictRe.FindStringSubmatch(trimmed); vm != nil {
					if vm[1] == "pass" {
						addLine(styleOK.Render("Verdict: pass"))
					} else {
						addLine(styleStop.Render("Verdict: fix"))
					}
					continue
				}
				if hm := headLineRe.FindStringSubmatch(trimmed); hm != nil {
					short := hm[1]
					if len(short) > 7 {
						short = short[:7]
					}
					addLine(styleMuted.Render("Reviewed commit: " + short))
					continue
				}
				if sm := sectionList.FindStringSubmatch(trimmed); sm != nil {
					label := "Notes:"
					if sm[1] == "fix" {
						label = "To fix:"
					}
					addLine(styleTitle.Render(label))
					continue
				}
				if sev := sevLineRe.FindStringSubmatch(rawLine); sev != nil {
					badgeStyle := styleMuted
					if sev[1] == "P0" || sev[1] == "P1" {
						badgeStyle = styleStop
					}
					wrapped := lipgloss.NewStyle().Width(maxInt(20, wrapWidth-6)).Render(sev[2])
					wLines := strings.Split(wrapped, "\n")
					addLine(badgeStyle.Render("["+sev[1]+"] ") + wLines[0])
					for _, cont := range wLines[1:] {
						addLine("     " + cont)
					}
					continue
				}
			}
			if strings.HasPrefix(rawLine, "### ") || strings.HasPrefix(rawLine, "## ") || strings.HasPrefix(rawLine, "# ") {
				addWrapped("", rawLine, styleTitle)
				continue
			}
			addWrapped("", rawLine, lipgloss.NewStyle())
		}
	}

	return strings.Join(lines, "\n"), sectionOffsets
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
