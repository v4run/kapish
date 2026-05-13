package capi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

func TestWatchClusters_ReceivesAddedEvent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clusterv1.AddToScheme(scheme))

	dyn := fake.NewSimpleDynamicClient(scheme)
	c := &Client{dyn: dyn, namespace: ""}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := c.WatchClusters(ctx)
	require.NoError(t, err)

	go func() {
		time.Sleep(50 * time.Millisecond)
		u := toUnstructured(t, newFakeCluster("new", "default", "Pending"))
		_, _ = dyn.Resource(clusterGVR).Namespace("default").Create(
			context.Background(), u, metav1.CreateOptions{},
		)
	}()

	select {
	case ev := <-events:
		assert.Equal(t, EventAdded, ev.Type)
		assert.Equal(t, "new", ev.Cluster.Name)
		assert.Equal(t, "default", ev.Cluster.Namespace)
	case <-ctx.Done():
		t.Fatalf("did not receive Added event before timeout")
	}
}

func toUnstructured(t *testing.T, obj any) *unstructured.Unstructured {
	t.Helper()
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: u}
}
