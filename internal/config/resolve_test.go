package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyOverrides_FlagBeatsFileBeatsDefaults(t *testing.T) {
	c, err := LoadFromFile("testdata/valid_basic.yaml")
	require.NoError(t, err)

	// No flags: file values stand.
	got := ApplyOverrides(c, FlagOverrides{})
	assert.Equal(t, "prod-mgmt", got.ManagementClusters.Current)
	assert.Equal(t, "/home/me/.kube/prod", got.ManagementClusters.Entries[0].Kubeconfig)

	// --kubeconfig and --context flags override the current entry.
	got = ApplyOverrides(c, FlagOverrides{
		Kubeconfig: "/tmp/override.kubeconfig",
		Context:    "override-ctx",
	})
	assert.Equal(t, "/tmp/override.kubeconfig", got.ManagementClusters.Entries[0].Kubeconfig)
	assert.Equal(t, "override-ctx", got.ManagementClusters.Entries[0].Context)
}

func TestApplyOverrides_OneShotFlag(t *testing.T) {
	c := Defaults()
	got := ApplyOverrides(c, FlagOverrides{OneShot: boolPtr(true)})
	assert.True(t, got.UI.OneShot)

	c.UI.OneShot = true
	got = ApplyOverrides(c, FlagOverrides{OneShot: boolPtr(false)})
	assert.False(t, got.UI.OneShot, "explicit false flag must override file's true")
}

// When no managementClusters.current is set but flags supply kubeconfig/context,
// a synthetic single-entry list is created so downstream code always has one.
func TestApplyOverrides_SynthesizesEntryWhenAbsent(t *testing.T) {
	c := Defaults()
	got := ApplyOverrides(c, FlagOverrides{
		Kubeconfig: "/tmp/k",
		Context:    "ctx",
	})
	require.Len(t, got.ManagementClusters.Entries, 1)
	assert.Equal(t, "default", got.ManagementClusters.Entries[0].Name)
	assert.Equal(t, "default", got.ManagementClusters.Current)
	assert.Equal(t, "/tmp/k", got.ManagementClusters.Entries[0].Kubeconfig)
	assert.Equal(t, "ctx", got.ManagementClusters.Entries[0].Context)
}

func boolPtr(b bool) *bool { return &b }

// When Current references a missing entry, ApplyOverrides must NOT silently
// re-point to entry[0]. Validate() should still see the misconfiguration.
func TestApplyOverrides_PreservesStaleCurrent(t *testing.T) {
	c := Defaults()
	c.ManagementClusters = ManagementClustersConfig{
		Current: "missing",
		Entries: []ManagementClusterEntry{{Name: "real"}},
	}

	got := ApplyOverrides(c, FlagOverrides{
		Kubeconfig: "/tmp/k",
		Context:    "ctx",
	})

	// Current preserved as-is (still bad).
	assert.Equal(t, "missing", got.ManagementClusters.Current)
	// No silent override of entries[0].
	assert.Equal(t, "", got.ManagementClusters.Entries[0].Kubeconfig)
	assert.Equal(t, "", got.ManagementClusters.Entries[0].Context)

	// Validate must fail on this — it's still a misconfiguration.
	err := Validate(got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "current")
}
