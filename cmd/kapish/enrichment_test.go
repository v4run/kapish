package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/enrich"
)

func TestBuildEnrichmentManager_Nil_WhenEmpty(t *testing.T) {
	assert.Nil(t, buildEnrichmentManager(kconfig.EnrichmentConfig{}))
}

func TestBuildEnrichmentManager_SkipsBadProvider(t *testing.T) {
	cfg := kconfig.EnrichmentConfig{
		Providers: []kconfig.EnrichProvider{
			{Name: "bad", Type: "http", Command: []string{"x"}, Key: "id", RefreshIntervalSec: 1, TimeoutSec: 1},
		},
		Columns: []kconfig.EnrichColumn{{Header: "X", Provider: "bad", Field: "f", JoinOn: "name"}},
	}
	// The only provider is invalid, so no columns survive -> nil manager.
	assert.Nil(t, buildEnrichmentManager(cfg))
}

func TestBuildEnrichmentManager_BuildsColumns(t *testing.T) {
	cfg := kconfig.EnrichmentConfig{
		Providers: []kconfig.EnrichProvider{
			{Name: "cost", Type: "shell", Command: []string{"echo", "[]"}, Key: "id", RefreshIntervalSec: 300, TimeoutSec: 5},
		},
		Columns: []kconfig.EnrichColumn{{Header: "COST", Provider: "cost", Field: "c", JoinOn: "name"}},
	}
	mgr := buildEnrichmentManager(cfg)
	require.NotNil(t, mgr)
	cols := mgr.Columns()
	require.Len(t, cols, 1)
	assert.Equal(t, "COST", cols[0].Header)

	// Cells resolvable (pending before any fetch).
	cells := mgr.Cells(enrich.ClusterRef{Name: "c1"})
	require.Len(t, cells, 1)
	assert.Equal(t, enrich.CellPending, cells[0].State)
}

func TestBuildEnrichmentManager_SkipsEmptyNameProvider(t *testing.T) {
	cfg := kconfig.EnrichmentConfig{
		Providers: []kconfig.EnrichProvider{
			{Name: "", Type: "shell", Command: []string{"echo", "[]"}, Key: "id", RefreshIntervalSec: 300, TimeoutSec: 5},
			{Name: "cost", Type: "shell", Command: []string{"echo", "[]"}, Key: "id", RefreshIntervalSec: 300, TimeoutSec: 5},
		},
		Columns: []kconfig.EnrichColumn{{Header: "COST", Provider: "cost", Field: "c", JoinOn: "name"}},
	}
	mgr := buildEnrichmentManager(cfg)
	require.NotNil(t, mgr)
	cols := mgr.Columns()
	require.Len(t, cols, 1)
	assert.Equal(t, "COST", cols[0].Header)
}

func TestBuildEnrichmentManager_SkipsDuplicateNameProvider(t *testing.T) {
	cfg := kconfig.EnrichmentConfig{
		Providers: []kconfig.EnrichProvider{
			{Name: "cost", Type: "shell", Command: []string{"echo", "[]"}, Key: "id", RefreshIntervalSec: 300, TimeoutSec: 5},
			{Name: "cost", Type: "shell", Command: []string{"echo", "[]"}, Key: "id", RefreshIntervalSec: 999, TimeoutSec: 99},
		},
		Columns: []kconfig.EnrichColumn{{Header: "COST", Provider: "cost", Field: "c", JoinOn: "name"}},
	}
	mgr := buildEnrichmentManager(cfg)
	require.NotNil(t, mgr)
	cols := mgr.Columns()
	require.Len(t, cols, 1, "only the first provider named \"cost\" should be kept")
	assert.Equal(t, "COST", cols[0].Header)

	// A column referencing "cost" still resolves against the first provider.
	cells := mgr.Cells(enrich.ClusterRef{Name: "c1"})
	require.Len(t, cells, 1)
	assert.Equal(t, enrich.CellPending, cells[0].State)
}
