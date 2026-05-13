package capi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

func TestListClusters_ReturnsAllNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clusterv1.AddToScheme(scheme))

	a := newFakeCluster("a", "ns1", "Provisioned")
	b := newFakeCluster("b", "ns2", "Pending")
	dyn := fake.NewSimpleDynamicClient(scheme, a, b)

	c := &Client{dyn: dyn, namespace: ""}

	got, err := c.ListClusters(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)

	byName := map[string]Cluster{}
	for _, x := range got {
		byName[x.Name] = x
	}
	assert.Equal(t, "ns1", byName["a"].Namespace)
	assert.Equal(t, "Provisioned", byName["a"].Phase)
	assert.Equal(t, "ns2", byName["b"].Namespace)
	assert.Equal(t, "Pending", byName["b"].Phase)
}

func TestListClusters_NamespaceScoped(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clusterv1.AddToScheme(scheme))

	a := newFakeCluster("a", "ns1", "Provisioned")
	b := newFakeCluster("b", "ns2", "Pending")
	dyn := fake.NewSimpleDynamicClient(scheme, a, b)

	c := &Client{dyn: dyn, namespace: "ns1"}

	got, err := c.ListClusters(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "a", got[0].Name)
}

func newFakeCluster(name, namespace, phase string) *clusterv1.Cluster {
	return &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now().UTC().Truncate(time.Second)),
		},
		Status: clusterv1.ClusterStatus{Phase: phase},
	}
}
