package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	kconfig "github.com/v4run/kapish/internal/config"
)

func (m Model) openSettings() (tea.Model, tea.Cmd) {
	m.screen = screenSettings
	return m, nil
}

func (m Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.screen = screenReady
		return m, nil
	case "t":
		if m.cfg.AppConfig.UI.Theme == "dark" {
			m.cfg.AppConfig.UI.Theme = "light"
		} else {
			m.cfg.AppConfig.UI.Theme = "dark"
		}
		return m, m.persistConfigCmd()
	}
	return m, nil
}

// persistConfigCmd writes the current AppConfig back to ConfigPath (no-op if
// ConfigPath is empty, e.g. in tests).
func (m Model) persistConfigCmd() tea.Cmd {
	if m.cfg.ConfigPath == "" {
		return nil
	}
	cfg := m.cfg.AppConfig
	path := m.cfg.ConfigPath
	return func() tea.Msg {
		_ = kconfig.WriteToFile(path, cfg) // best-effort; surfacing errors is a follow-up
		return nil
	}
}
