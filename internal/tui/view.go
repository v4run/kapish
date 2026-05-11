package tui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	switch m.screen {
	case screenLoading:
		return m.viewCentered("Loading clusters…")
	case screenError:
		return m.viewError()
	case screenSettings:
		return m.viewSettings() // Task 10 replaces
	case screenMgmtPicker:
		return m.viewMgmtPicker() // Task 9 replaces
	case screenSpawning:
		return m.viewCentered("Starting shell…")
	default:
		return m.viewList()
	}
}

func (m Model) header() string {
	return styleTitle.Render(fmt.Sprintf("kapish — mgmt: %s (%d clusters)", m.mgmtContext, len(m.clusters)))
}

func (m Model) viewList() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	if m.filter.Focused() || m.filter.Value() != "" {
		b.WriteString(m.filter.View() + "\n")
	}
	if len(m.filtered) == 0 {
		if m.filter.Value() != "" {
			b.WriteString("\n  No clusters match filter.\n")
		} else {
			b.WriteString("\n  No CAPI clusters found on " + m.mgmtContext + ". Press 'r' to refresh.\n")
		}
	} else {
		b.WriteString(styleDim.Render(fmt.Sprintf("  %-28s %-12s %-14s %-9s %s", "NAME", "NS", "PHASE", "VERSION", "PROVIDER")) + "\n")
		for i, c := range m.filtered {
			cursor := "  "
			name := c.Name
			if i == m.cursor {
				cursor = styleSelect.Render("▸ ")
				name = styleSelect.Render(c.Name)
			}
			ver := c.K8sVersion
			if ver == "" {
				ver = "-"
			}
			prov := c.Provider
			if prov == "" {
				prov = "-"
			}
			b.WriteString(fmt.Sprintf("%s%-28s %-12s %-14s %-9s %s %s\n",
				cursor, name, c.Namespace, phaseStyled(c.Phase), ver, prov, phaseGlyph(c.Phase)))
		}
	}
	if m.confirmingSpawn {
		b.WriteString("\n" + styleRed.Render(fmt.Sprintf(
			"Cluster %s is %s. Spawn shell anyway? (y/N)",
			m.spawnTarget.Name, m.spawnTarget.Phase,
		)))
	}
	b.WriteString("\n" + styleStatus.Render("↑↓ nav · / filter · ⏎ shell · r refresh · m mgmt · s settings · q quit"))
	return b.String()
}

func (m Model) viewCentered(s string) string {
	return "\n  " + s + "\n"
}

func (m Model) viewError() string {
	msg := "unknown error"
	if m.err != nil {
		msg = m.err.Error()
	}
	return fmt.Sprintf("\n  %s\n\n  %s\n\n  %s\n",
		styleTitle.Render("kapish — error"),
		styleRed.Render(msg),
		styleStatus.Render("r retry · m mgmt · q quit"))
}

// Placeholders replaced by later tasks. TODO(plan3): Task 10.
func (m Model) viewSettings() string { return m.viewCentered("settings (TODO Task 10)") }

func (m Model) viewMgmtPicker() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("kapish — switch management cluster") + "\n\n")
	entries := m.cfg.AppConfig.ManagementClusters.Entries
	current := m.cfg.AppConfig.ManagementClusters.Current
	if len(entries) == 0 {
		b.WriteString(styleDim.Render("  No management clusters configured.\n"))
	} else {
		for i, e := range entries {
			cursor := "  "
			name := e.Name
			if i == m.mgmtCursor {
				cursor = styleSelect.Render("▸ ")
				name = styleSelect.Render(e.Name)
			}
			suffix := ""
			if e.Name == current {
				suffix = styleDim.Render(" (current)")
			}
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, name, suffix))
		}
	}
	b.WriteString("\n" + styleStatus.Render("↑↓ select · ⏎ switch · esc cancel"))
	return b.String()
}
