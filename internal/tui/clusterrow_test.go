package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestSortClusters_StableByNamespaceThenName(t *testing.T) {
	in := []capi.Cluster{
		{Name: "b", Namespace: "ns2"},
		{Name: "a", Namespace: "ns2"},
		{Name: "z", Namespace: "ns1"},
	}
	out := sortClusters(in)
	require.Len(t, out, 3)
	assert.Equal(t, "ns1/z", out[0].Namespace+"/"+out[0].Name)
	assert.Equal(t, "ns2/a", out[1].Namespace+"/"+out[1].Name)
	assert.Equal(t, "ns2/b", out[2].Namespace+"/"+out[2].Name)
}

func TestClusterKey(t *testing.T) {
	c := capi.Cluster{Name: "x", Namespace: "ns"}
	assert.Equal(t, "ns/x", clusterKey(c))
}

func TestFilterClusters_FuzzyMatchNameAndNamespace(t *testing.T) {
	in := []capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod"},
		{Name: "stg-eu-1", Namespace: "staging"},
		{Name: "prod-us-1", Namespace: "prod"},
	}
	got := filterClusters(in, "prod")
	require.Len(t, got, 2)
	for _, c := range got {
		assert.Contains(t, c.Name+c.Namespace, "prod")
	}
	assert.Len(t, filterClusters(in, ""), 3)
	got = filterClusters(in, "staging")
	require.Len(t, got, 1)
	assert.Equal(t, "stg-eu-1", got[0].Name)
}

func TestAgeString(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		created time.Time
		want    string
	}{
		{now.Add(-30 * time.Second), "30s"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-49 * time.Hour), "2d"},
		{time.Time{}, "?"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ageString(tc.created, now), "created=%v", tc.created)
	}
}
