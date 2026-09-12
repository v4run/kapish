package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/v4run/kapish/internal/enrich"
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

// displayWidth is the rendered column width of s, ignoring ANSI escapes.
func displayWidth(s string) int { return lipgloss.Width(s) }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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

// enrichCellText is the plain (unstyled) text an enrichment cell occupies —
// used both for width measurement and for the Ready rendering.
func enrichCellText(c enrich.Cell) string {
	switch c.State {
	case enrich.CellReady:
		return c.Text
	case enrich.CellError:
		return "!"
	case enrich.CellEmpty:
		return "-"
	default: // CellPending
		return "…"
	}
}

// enrichCellRender pads the cell to width w and applies its state styling.
func enrichCellRender(c enrich.Cell, w int) string {
	text := padRight(enrichCellText(c), w)
	switch c.State {
	case enrich.CellReady:
		return text
	case enrich.CellError:
		return styleRed.Render(text)
	default: // Empty, Pending
		return styleDim.Render(text)
	}
}

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
