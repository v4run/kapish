package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// padRight pads s with trailing spaces to the given display width. It measures
// visible width (ignoring any ANSI escape sequences), so it must be applied to
// plain text before styling — padding an already-styled string counts the
// escape bytes and misaligns the column.
func padRight(s string, w int) string {
	if pad := w - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

var (
	styleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleTitle  = lipgloss.NewStyle().Bold(true)
	styleSelect = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	styleStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func phaseGlyph(phase string) string {
	switch phase {
	case "Provisioned":
		return "✓"
	case "Provisioning", "Pending":
		return "…"
	case "Failed", "Deleting":
		return "⚠"
	default:
		return "·"
	}
}

func phaseStyled(phase string) string {
	return phaseCell(phase, 0)
}

// phaseCell renders the phase padded to width w (0 = no padding) and colored by
// phase. Padding happens on the plain text before styling so the column aligns.
func phaseCell(phase string, w int) string {
	text := phase
	if text == "" {
		text = "-"
	}
	text = padRight(text, w)
	switch phase {
	case "Provisioned":
		return styleGreen.Render(text)
	case "Provisioning", "Pending":
		return styleYellow.Render(text)
	case "Failed", "Deleting":
		return styleRed.Render(text)
	default:
		return styleDim.Render(text)
	}
}
