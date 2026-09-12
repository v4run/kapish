package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/v4run/kapish/internal/enrich"
)

func (m Model) View() string {
	switch m.screen {
	case screenLoading:
		return m.viewCentered("Loading clusters…")
	case screenError:
		return m.viewError()
	case screenSettings:
		return m.viewSettings()
	case screenMgmtPicker:
		return m.viewMgmtPicker()
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
		// Enrichment columns (may be none).
		var ecols []enrich.Column
		if mgr := m.cfg.Enrichment; mgr != nil {
			ecols = mgr.Columns()
		}

		// Precompute per-row enrichment cells and column widths.
		ecW := make([]int, len(ecols))
		for i, col := range ecols {
			ecW[i] = maxInt(displayWidth(col.Header), col.Width)
		}
		rowCells := make([][]enrich.Cell, len(m.filtered))
		ageW := displayWidth("AGE")
		now := time.Now()
		for r, c := range m.filtered {
			ageW = maxInt(ageW, displayWidth(ageString(c.CreationTimestamp, now)))
			if mgr := m.cfg.Enrichment; mgr != nil {
				cells := mgr.Cells(enrich.ClusterRef{Name: c.Name, Namespace: c.Namespace})
				rowCells[r] = cells
				for i := range ecols {
					ecW[i] = maxInt(ecW[i], displayWidth(enrichCellText(cells[i])))
				}
			}
		}

		// Dynamic name/ns widths (unchanged behavior).
		nameW := displayWidth("NAME")
		nsW := displayWidth("NS")
		for _, c := range m.filtered {
			nameW = maxInt(nameW, displayWidth(c.Name))
			nsW = maxInt(nsW, displayWidth(c.Namespace))
		}

		// Header.
		header := fmt.Sprintf("  %s %s %-14s %-9s %-10s %s",
			padRight("NAME", nameW), padRight("NS", nsW), "PHASE", "VERSION", "PROVIDER", padRight("AGE", ageW))
		for i, col := range ecols {
			header += " " + padRight(col.Header, ecW[i])
		}
		b.WriteString(styleDim.Render(header) + "\n")

		// Rows.
		for i, c := range m.filtered {
			cursor := "  "
			name := padRight(c.Name, nameW)
			if i == m.cursor {
				cursor = styleSelect.Render("▸ ")
				name = styleSelect.Render(name)
			}
			ver := c.K8sVersion
			if ver == "" {
				ver = "-"
			}
			prov := c.Provider
			if prov == "" {
				prov = "-"
			}
			line := fmt.Sprintf("%s%s %s %s %s %s %s",
				cursor, name, padRight(c.Namespace, nsW), phaseCell(c.Phase, 14),
				padRight(ver, 9), padRight(prov, 10), padRight(ageString(c.CreationTimestamp, now), ageW))
			for j := range ecols {
				line += " " + enrichCellRender(rowCells[i][j], ecW[j])
			}
			line += " " + phaseGlyph(c.Phase)
			b.WriteString(line + "\n")
		}

		// Explain the selected row's enrichment errors, if any.
		if mgr := m.cfg.Enrichment; mgr != nil && m.cursor >= 0 && m.cursor < len(rowCells) {
			perr := mgr.ProviderErrors()
			for j, col := range ecols {
				if rowCells[m.cursor][j].State == enrich.CellError {
					if e := perr[col.Provider]; e != nil {
						b.WriteString(styleRed.Render("  "+col.Header+": "+e.Error()) + "\n")
						break
					}
				}
			}
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

func (m Model) viewSettings() string {
	c := m.cfg.AppConfig
	var b strings.Builder
	b.WriteString(styleTitle.Render("kapish — settings") + "\n\n")
	b.WriteString(fmt.Sprintf("  shell command:       %s\n", c.Shell.Command))
	b.WriteString(fmt.Sprintf("  cwd:                 %s\n", c.Shell.Cwd))
	b.WriteString(fmt.Sprintf("  env count:           %d\n", len(c.Shell.Env)))
	b.WriteString(fmt.Sprintf("  alias count:         %d\n", len(c.Shell.Aliases)))
	b.WriteString(fmt.Sprintf("  prompt:              %s\n", c.Shell.Prompt))
	b.WriteString(fmt.Sprintf("  theme:               %s\n", c.UI.Theme))
	b.WriteString(fmt.Sprintf("  refreshIntervalSec:  %d\n", c.UI.RefreshIntervalSec))
	b.WriteString(fmt.Sprintf("  oneShot:             %v\n", c.UI.OneShot))
	b.WriteString(fmt.Sprintf("  web port:            %d\n", c.Web.DefaultPort))
	b.WriteString(fmt.Sprintf("  web openBrowser:     %v\n", c.Web.OpenBrowser))
	b.WriteString(fmt.Sprintf("  web bindAddr:        %s\n", c.Web.BindAddr))
	b.WriteString(fmt.Sprintf("  mgmt cluster count:  %d\n", len(c.ManagementClusters.Entries)))
	b.WriteString("\n" + styleStatus.Render("t toggle theme · esc back"))
	return b.String()
}

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
