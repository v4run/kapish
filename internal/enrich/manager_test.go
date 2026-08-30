package enrich

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// erroringProvider always fails its Fetch.
type erroringProvider struct{ name string }

func (p erroringProvider) Name() string { return p.name }
func (p erroringProvider) Fetch(context.Context) (map[string]Row, error) {
	return nil, context.DeadlineExceeded
}

func testManager() *Manager {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("cost", map[string]Row{
			"prod-eu-1": {"monthlyCost": "$412", "nodes": "12"},
		}),
		Interval: time.Minute,
		Timeout:  time.Second,
	}}
	cols := []Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	return NewManager(specs, cols)
}

func TestCells_PendingBeforeFetch(t *testing.T) {
	m := testManager()
	cells := m.Cells(ClusterRef{Name: "prod-eu-1"})
	require.Len(t, cells, 1)
	assert.Equal(t, CellPending, cells[0].State)
}

func TestCells_ReadyAfterFetch(t *testing.T) {
	m := testManager()
	m.refreshOnce(context.Background(), "cost")
	cells := m.Cells(ClusterRef{Name: "prod-eu-1"})
	assert.Equal(t, CellReady, cells[0].State)
	assert.Equal(t, "$412", cells[0].Text)
}

func TestCells_EmptyWhenNoMatch(t *testing.T) {
	m := testManager()
	m.refreshOnce(context.Background(), "cost")
	cells := m.Cells(ClusterRef{Name: "does-not-exist"})
	assert.Equal(t, CellEmpty, cells[0].State)
}

func TestCells_ErrorWhenFirstFetchFails(t *testing.T) {
	specs := []ProviderSpec{{Provider: erroringProvider{name: "cost"}, Interval: time.Minute, Timeout: time.Second}}
	cols := []Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	m := NewManager(specs, cols)
	m.refreshOnce(context.Background(), "cost")
	cells := m.Cells(ClusterRef{Name: "prod-eu-1"})
	assert.Equal(t, CellError, cells[0].State)
	assert.NotNil(t, m.ProviderErrors()["cost"])
}

func TestCells_JoinOnNamespace(t *testing.T) {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("team", map[string]Row{"prod": {"owner": "platform"}}),
		Interval: time.Minute, Timeout: time.Second,
	}}
	cols := []Column{{Header: "OWNER", Provider: "team", Field: "owner", JoinOn: "namespace"}}
	m := NewManager(specs, cols)
	m.refreshOnce(context.Background(), "team")
	cells := m.Cells(ClusterRef{Name: "prod-eu-1", Namespace: "prod"})
	assert.Equal(t, CellReady, cells[0].State)
	assert.Equal(t, "platform", cells[0].Text)
}
