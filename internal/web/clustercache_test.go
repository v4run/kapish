package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestClusterCache_SnapshotReturnsSortedCopy(t *testing.T) {
	c := newClusterCache()
	c.replaceAll([]capi.Cluster{
		{Name: "b", Namespace: "ns"},
		{Name: "a", Namespace: "ns"},
	})
	snap := c.snapshot()
	require.Len(t, snap, 2)
	assert.Equal(t, "a", snap[0].Name)
	assert.Equal(t, "b", snap[1].Name)

	// Mutating the returned slice must not affect the cache.
	snap[0].Name = "MUTATED"
	assert.Equal(t, "a", c.snapshot()[0].Name)
}

func TestClusterCache_ApplyEventAddModifyDelete(t *testing.T) {
	c := newClusterCache()
	c.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "x", Namespace: "ns", Phase: "Pending"}})
	require.Len(t, c.snapshot(), 1)
	assert.Equal(t, "Pending", c.snapshot()[0].Phase)

	c.applyEvent(capi.Event{Type: capi.EventModified, Cluster: capi.Cluster{Name: "x", Namespace: "ns", Phase: "Provisioned"}})
	assert.Equal(t, "Provisioned", c.snapshot()[0].Phase)

	c.applyEvent(capi.Event{Type: capi.EventDeleted, Cluster: capi.Cluster{Name: "x", Namespace: "ns"}})
	assert.Empty(t, c.snapshot())
}

func TestClusterCache_SubscribeReceivesEvents(t *testing.T) {
	c := newClusterCache()
	sub, unsub := c.subscribe()
	defer unsub()

	go c.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "y", Namespace: "ns"}})

	select {
	case ev := <-sub:
		assert.Equal(t, capi.EventAdded, ev.Type)
		assert.Equal(t, "y", ev.Cluster.Name)
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event")
	}
}

func TestClusterCache_UnsubStopsDelivery(t *testing.T) {
	c := newClusterCache()
	sub, unsub := c.subscribe()
	unsub()
	// applyEvent must not panic on a closed/removed subscriber.
	c.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "z", Namespace: "ns"}})
	// Channel should be closed; a receive returns zero-value + ok==false eventually.
	select {
	case _, ok := <-sub:
		assert.False(t, ok, "unsubbed channel should be closed")
	case <-time.After(time.Second):
		// also acceptable if implementation just stops sending
	}
}
