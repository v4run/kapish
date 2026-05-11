package tui

import "github.com/charmbracelet/lipgloss"

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
	switch phase {
	case "Provisioned":
		return styleGreen.Render(phase)
	case "Provisioning", "Pending":
		return styleYellow.Render(phase)
	case "Failed", "Deleting":
		return styleRed.Render(phase)
	default:
		if phase == "" {
			return styleDim.Render("-")
		}
		return styleDim.Render(phase)
	}
}
