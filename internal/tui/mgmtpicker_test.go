package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kconfig "github.com/v4run/kapish/internal/config"
)

func modelWithMgmtEntries(current string, names ...string) Model {
	entries := make([]kconfig.ManagementClusterEntry, 0, len(names))
	for _, n := range names {
		entries = append(entries, kconfig.ManagementClusterEntry{Name: n})
	}
	cfg := Config{
		AppConfig: kconfig.Config{
			ManagementClusters: kconfig.ManagementClustersConfig{Current: current, Entries: entries},
		},
		MgmtContext: current,
	}
	m := New(cfg)
	mu, _ := m.Update(clustersLoadedMsg{})
	return mu.(Model)
}

func TestMgmtPicker_OpensListsEntriesHighlightsCurrent(t *testing.T) {
	m := modelWithMgmtEntries("b", "a", "b", "c")
	mu, _ := m.Update(key('m'))
	m = mu.(Model)
	require.Equal(t, screenMgmtPicker, m.screen)
	// Cursor starts on the current entry ("b" = index 1).
	assert.Equal(t, 1, m.mgmtCursor)
}

func TestMgmtPicker_EscCancels(t *testing.T) {
	m := modelWithMgmtEntries("a", "a", "b")
	mu, _ := m.Update(key('m'))
	m = mu.(Model)
	mu, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mu.(Model)
	assert.Equal(t, screenReady, m.screen)
}

func TestMgmtPicker_EnterEmitsSwitchIntent(t *testing.T) {
	m := modelWithMgmtEntries("a", "a", "b")
	mu, _ := m.Update(key('m'))
	m = mu.(Model)
	mu, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // move to "b"
	m = mu.(Model)
	mu, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mu.(Model)
	// We can't run the cmd (it rebuilds a client); assert the screen left the
	// picker and a cmd is returned.
	assert.NotEqual(t, screenMgmtPicker, m.screen)
	assert.NotNil(t, cmd)
}
