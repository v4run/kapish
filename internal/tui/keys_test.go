package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func readyModelWith(clusters ...capi.Cluster) Model {
	m := New(Config{})
	updated, _ := m.Update(clustersLoadedMsg{clusters: clusters})
	return updated.(Model)
}

func key(r rune) tea.KeyMsg            { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
func special(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func TestNav_DownUpClampsAtEnds(t *testing.T) {
	m := readyModelWith(
		capi.Cluster{Name: "a", Namespace: "ns"},
		capi.Cluster{Name: "b", Namespace: "ns"},
		capi.Cluster{Name: "c", Namespace: "ns"},
	)
	assert.Equal(t, 0, m.cursor)

	m, _ = updateKey(m, key('j'))
	m, _ = updateKey(m, key('j'))
	assert.Equal(t, 2, m.cursor)
	m, _ = updateKey(m, key('j')) // clamps
	assert.Equal(t, 2, m.cursor)

	m, _ = updateKey(m, key('k'))
	assert.Equal(t, 1, m.cursor)
	m, _ = updateKey(m, special(tea.KeyUp))
	m, _ = updateKey(m, special(tea.KeyUp)) // clamps
	assert.Equal(t, 0, m.cursor)
}

func TestNav_GAndShiftG(t *testing.T) {
	m := readyModelWith(
		capi.Cluster{Name: "a", Namespace: "ns"},
		capi.Cluster{Name: "b", Namespace: "ns"},
		capi.Cluster{Name: "c", Namespace: "ns"},
	)
	m, _ = updateKey(m, key('G'))
	assert.Equal(t, 2, m.cursor)
	m, _ = updateKey(m, key('g'))
	assert.Equal(t, 0, m.cursor)
}

func TestFilterMode_SlashEntersFilterTypingFiltersEscCancels(t *testing.T) {
	m := readyModelWith(
		capi.Cluster{Name: "prod-1", Namespace: "prod"},
		capi.Cluster{Name: "stg-1", Namespace: "staging"},
	)
	m, _ = updateKey(m, key('/'))
	assert.True(t, m.filter.Focused())

	m, _ = updateKey(m, key('p'))
	m, _ = updateKey(m, key('r'))
	require.Len(t, m.filtered, 1)
	assert.Equal(t, "prod-1", m.filtered[0].Name)

	// Esc clears filter + unfocuses.
	m, _ = updateKey(m, special(tea.KeyEsc))
	assert.False(t, m.filter.Focused())
	assert.Equal(t, "", m.filter.Value())
	assert.Len(t, m.filtered, 2)
}

func TestQuitKeys(t *testing.T) {
	m := readyModelWith(capi.Cluster{Name: "a", Namespace: "ns"})
	_, cmd := updateKey(m, key('q'))
	assert.NotNil(t, cmd, "q should return a quit cmd")
	// We can't easily assert it IS tea.Quit, but a non-nil cmd is the signal.

	_, cmd = updateKey(m, special(tea.KeyCtrlC))
	assert.NotNil(t, cmd)
}

// updateKey is a tiny test helper.
func updateKey(m Model, k tea.KeyMsg) (Model, tea.Cmd) {
	updated, cmd := m.Update(k)
	return updated.(Model), cmd
}
