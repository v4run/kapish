package enrich

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart_FillsCellsAndSignals(t *testing.T) {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("cost", map[string]Row{"c1": {"v": "42"}}),
		Interval: time.Hour, // only the immediate first fetch matters here
		Timeout:  time.Second,
	}}
	cols := []Column{{Header: "V", Provider: "cost", Field: "v", JoinOn: "name"}}
	m := NewManager(specs, cols)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	select {
	case <-m.Updates():
	case <-time.After(2 * time.Second):
		t.Fatal("expected an update signal after Start")
	}

	cells := m.Cells(ClusterRef{Name: "c1"})
	require.Len(t, cells, 1)
	assert.Equal(t, CellReady, cells[0].State)
	assert.Equal(t, "42", cells[0].Text)
}

func TestRefreshAll(t *testing.T) {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("cost", map[string]Row{"c1": {"v": "9"}}),
		Interval: time.Hour, Timeout: time.Second,
	}}
	cols := []Column{{Header: "V", Provider: "cost", Field: "v", JoinOn: "name"}}
	m := NewManager(specs, cols)
	m.RefreshAll(context.Background())
	assert.Equal(t, CellReady, m.Cells(ClusterRef{Name: "c1"})[0].State)
}
