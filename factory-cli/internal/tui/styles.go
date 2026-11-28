package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorMuted  = lipgloss.AdaptiveColor{Light: "#66758A", Dark: "#8B98AB"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#C7D2E0", Dark: "#394659"}
	colorInk    = lipgloss.AdaptiveColor{Light: "#17263D", Dark: "#E6EDF7"}
	colorGate   = lipgloss.AdaptiveColor{Light: "#A24C0A", Dark: "#F59E0B"}
	colorOK     = lipgloss.AdaptiveColor{Light: "#2A6F50", Dark: "#34D399"}
	colorStop    = lipgloss.AdaptiveColor{Light: "#A8352A", Dark: "#F87171"}
	colorAccent  = lipgloss.AdaptiveColor{Light: "#1E4B8C", Dark: "#60A5FA"}
	colorRunning = lipgloss.AdaptiveColor{Light: "#0D7490", Dark: "#22D3EE"}
	colorSelBg   = lipgloss.AdaptiveColor{Light: "#E8EFF8", Dark: "#243247"}
	colorCodeBg  = lipgloss.AdaptiveColor{Light: "#EEF3F9", Dark: "#1E293B"}

	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(colorInk)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleGate    = lipgloss.NewStyle().Bold(true).Foreground(colorGate)
	styleOK      = lipgloss.NewStyle().Bold(true).Foreground(colorOK)
	styleStop    = lipgloss.NewStyle().Bold(true).Foreground(colorStop)
	stylePlan    = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleRunning = lipgloss.NewStyle().Bold(true).Foreground(colorRunning)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorInk).
			Background(colorSelBg).
			Padding(0, 1)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 1)

	styleTabGate = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGate).
			Padding(0, 1)

	styleSelectedRow = lipgloss.NewStyle().
				Background(colorSelBg).
				Foreground(colorInk)

	styleDivider = lipgloss.NewStyle().Foreground(colorBorder)

	styleCodeBlock = lipgloss.NewStyle().
			Background(colorCodeBg).
			Foreground(colorInk).
			Padding(0, 1)

	styleHeaderBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorBorder)

	styleFooterBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colorBorder).
			Foreground(colorMuted)
)
