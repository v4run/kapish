package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultsMatchSpec(t *testing.T) {
	d := Defaults()

	// management clusters: empty entries, empty current
	assert.Equal(t, "", d.ManagementClusters.Current)
	assert.Empty(t, d.ManagementClusters.Entries)

	// shell defaults
	assert.Equal(t, "", d.Shell.Command, "Shell.Command default empty -> resolves to $SHELL at spawn")
	assert.Equal(t, "", d.Shell.Cwd)
	assert.Equal(t, "[{cluster}] ", d.Shell.Prompt)
	assert.NotNil(t, d.Shell.Env, "Env must be a non-nil map")
	assert.Empty(t, d.Shell.Env)
	assert.NotNil(t, d.Shell.Aliases)
	assert.Empty(t, d.Shell.Aliases)

	// ui defaults
	assert.Equal(t, "dark", d.UI.Theme)
	assert.Equal(t, 30, d.UI.RefreshIntervalSec)
	assert.False(t, d.UI.OneShot)

	// web defaults
	assert.Equal(t, 0, d.Web.DefaultPort)
	assert.True(t, d.Web.OpenBrowser)
	assert.Equal(t, "127.0.0.1", d.Web.BindAddr)
}

func TestDefaultsAreFresh(t *testing.T) {
	// Mutating one returned Defaults() must not affect the next.
	a := Defaults()
	a.Shell.Env["X"] = "1"
	a.Shell.Aliases["k"] = "kubectl"
	a.ManagementClusters.Entries = append(a.ManagementClusters.Entries, ManagementClusterEntry{Name: "x"})

	b := Defaults()
	require.Empty(t, b.Shell.Env, "second Defaults() must have empty Env")
	require.Empty(t, b.Shell.Aliases)
	require.Empty(t, b.ManagementClusters.Entries)
}
