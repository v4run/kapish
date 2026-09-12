package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/v4run/kapish/internal/capi"
	"github.com/v4run/kapish/internal/enrich"
)

func TestView_ListShowsClustersAndHeader(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu"})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = mu.(Model)
	mu, _ = m.Update(clustersLoadedMsg{clusters: []capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned", K8sVersion: "v1.30.2", Provider: "aws"},
		{Name: "stg-1", Namespace: "staging", Phase: "Failed", Provider: "gcp"},
	}})
	m = mu.(Model)

	out := m.View()
	assert.Contains(t, out, "mgmt-eu", "header shows mgmt context")
	assert.Contains(t, out, "prod-eu-1")
	assert.Contains(t, out, "stg-1")
	assert.Contains(t, out, "Provisioned")
	assert.Contains(t, out, "Failed")
	// Key hints in the status bar.
	assert.True(t, strings.Contains(out, "filter") && strings.Contains(out, "quit"))
}

func TestView_EmptyState(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu"})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mu.(Model)
	mu, _ = m.Update(clustersLoadedMsg{clusters: nil})
	m = mu.(Model)

	out := m.View()
	assert.Contains(t, out, "No CAPI clusters")
}

func TestView_LoadingState(t *testing.T) {
	m := New(Config{})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mu.(Model)
	out := m.View()
	assert.Contains(t, strings.ToLower(out), "loading")
}

func TestView_ErrorState(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu"})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mu.(Model)
	mu, _ = m.Update(errMsg{err: assertErr{}})
	m = mu.(Model)
	out := m.View()
	assert.Contains(t, out, "boom")
	assert.Contains(t, strings.ToLower(out), "retry")
}

func TestView_EnrichmentColumns(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu", Enrichment: readyManager()})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = mu.(Model)
	mu, _ = m.Update(clustersLoadedMsg{clusters: []capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned", K8sVersion: "v1.30.2", Provider: "aws"},
		{Name: "other", Namespace: "prod", Phase: "Failed", Provider: "gcp"},
	}})
	m = mu.(Model)

	out := m.View()
	assert.Contains(t, out, "COST", "enrichment header shows")
	assert.Contains(t, out, "$412", "joined value shows for matching cluster")
	// "other" has no matching row -> empty marker rendered somewhere.
	assert.Contains(t, out, "-")
}

func TestView_EnrichmentPending(t *testing.T) {
	// Manager with a provider that has never fetched -> pending marker.
	specs := []enrich.ProviderSpec{{
		Provider: enrich.NewStaticProvider("cost", map[string]enrich.Row{"prod-eu-1": {"monthlyCost": "$1"}}),
		Interval: time.Hour, Timeout: time.Second,
	}}
	cols := []enrich.Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	mgr := enrich.NewManager(specs, cols) // NOT refreshed

	m := New(Config{MgmtContext: "mgmt-eu", Enrichment: mgr})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = mu.(Model)
	mu, _ = m.Update(clustersLoadedMsg{clusters: []capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned"},
	}})
	m = mu.(Model)
	assert.Contains(t, m.View(), "…", "pending marker shows before first fetch")
}
