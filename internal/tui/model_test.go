package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestNewModel_StartsInLoading(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu"})
	assert.Equal(t, screenLoading, m.screen)
	assert.Equal(t, "mgmt-eu", m.mgmtContext)
}

func TestModel_ClustersLoadedMsgPopulatesAndGoesReady(t *testing.T) {
	m := New(Config{})
	updated, _ := m.Update(clustersLoadedMsg{clusters: []capi.Cluster{
		{Name: "b", Namespace: "ns"},
		{Name: "a", Namespace: "ns"},
	}})
	mm := updated.(Model)
	assert.Equal(t, screenReady, mm.screen)
	require.Len(t, mm.clusters, 2)
	// Sorted: a before b.
	assert.Equal(t, "a", mm.clusters[0].Name)
}

func TestModel_ErrMsgGoesToErrorScreen(t *testing.T) {
	m := New(Config{})
	updated, _ := m.Update(errMsg{err: assertErr{}})
	mm := updated.(Model)
	assert.Equal(t, screenError, mm.screen)
	assert.Error(t, mm.err)
}

func TestModel_WindowSizeStored(t *testing.T) {
	m := New(Config{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mm := updated.(Model)
	assert.Equal(t, 120, mm.width)
	assert.Equal(t, 40, mm.height)
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }
