package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/kapish/internal/capi"
	kconfig "github.com/v4run/kapish/internal/config"
)

// mgmtSwitchedMsg is sent after a successful mgmt-cluster switch.
type mgmtSwitchedMsg struct {
	client      *capi.Client
	mgmtContext string
	entryName   string
}

func (m Model) openMgmtPicker() (tea.Model, tea.Cmd) {
	m.screen = screenMgmtPicker
	m.mgmtCursor = 0
	cur := m.cfg.AppConfig.ManagementClusters.Current
	for i, e := range m.cfg.AppConfig.ManagementClusters.Entries {
		if e.Name == cur {
			m.mgmtCursor = i
			break
		}
	}
	return m, nil
}

func (m Model) handleMgmtPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.cfg.AppConfig.ManagementClusters.Entries
	switch msg.String() {
	case "esc", "q":
		m.screen = screenReady
		return m, nil
	case "up", "k":
		if m.mgmtCursor > 0 {
			m.mgmtCursor--
		}
		return m, nil
	case "down", "j":
		if m.mgmtCursor < len(entries)-1 {
			m.mgmtCursor++
		}
		return m, nil
	case "enter":
		if m.mgmtCursor < 0 || m.mgmtCursor >= len(entries) {
			m.screen = screenReady
			return m, nil
		}
		entry := entries[m.mgmtCursor]
		m.screen = screenLoading
		return m, m.switchMgmtCmd(entry)
	}
	return m, nil
}

func (m Model) switchMgmtCmd(entry kconfig.ManagementClusterEntry) tea.Cmd {
	return func() tea.Msg {
		c, err := capi.NewClient(capi.Options{
			Kubeconfig: entry.Kubeconfig,
			Context:    entry.Context,
			Namespace:  entry.Namespace,
		})
		if err != nil {
			return errMsg{err: err}
		}
		// Probe with a list so an unreachable mgmt fails fast.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err = c.ListClusters(ctx)
		cancel()
		if err != nil {
			return errMsg{err: err}
		}
		return mgmtSwitchedMsg{client: c, mgmtContext: c.Context(), entryName: entry.Name}
	}
}
