// Package web implements kapish's localhost HTTP+WebSocket server (kapish serve).
package web

import (
	"sort"
	"sync"

	"github.com/v4run/kapish/internal/capi"
)

// clusterCache holds the current set of CAPI clusters and fans events out to
// SSE subscribers. Safe for concurrent use.
type clusterCache struct {
	mu        sync.Mutex
	byKey     map[string]capi.Cluster
	subs      map[int]chan capi.Event
	nextSubID int
}

func newClusterCache() *clusterCache {
	return &clusterCache{byKey: map[string]capi.Cluster{}, subs: map[int]chan capi.Event{}}
}

func key(c capi.Cluster) string { return c.Namespace + "/" + c.Name }

// replaceAll resets the cache to exactly the given clusters (used after a LIST).
func (c *clusterCache) replaceAll(clusters []capi.Cluster) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byKey = make(map[string]capi.Cluster, len(clusters))
	for _, cl := range clusters {
		c.byKey[key(cl)] = cl
	}
}

// snapshot returns a sorted (namespace, name) copy of the current clusters.
func (c *clusterCache) snapshot() []capi.Cluster {
	c.mu.Lock()
	out := make([]capi.Cluster, 0, len(c.byKey))
	for _, cl := range c.byKey {
		out = append(out, cl)
	}
	c.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// applyEvent updates the cache and notifies subscribers (non-blocking; a slow
// subscriber may drop an event — SSE clients re-fetch the snapshot on connect).
// The fan-out runs inside mu so that unsub cannot close a channel concurrently
// with a send (sending on a closed channel panics even with select/default).
// Each send is non-blocking (default branch), so holding mu during the loop
// cannot deadlock.
func (c *clusterCache) applyEvent(ev capi.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch ev.Type {
	case capi.EventAdded, capi.EventModified:
		c.byKey[key(ev.Cluster)] = ev.Cluster
	case capi.EventDeleted:
		delete(c.byKey, key(ev.Cluster))
	case capi.EventError:
		return
	}
	for _, ch := range c.subs {
		select {
		case ch <- ev:
		default: // drop on slow subscriber
		}
	}
}

// subscribe returns a receive-only event channel and an unsubscribe func.
// The channel is closed by unsub.
func (c *clusterCache) subscribe() (<-chan capi.Event, func()) {
	c.mu.Lock()
	id := c.nextSubID
	c.nextSubID++
	ch := make(chan capi.Event, 32)
	c.subs[id] = ch
	c.mu.Unlock()
	return ch, func() {
		c.mu.Lock()
		if cur, ok := c.subs[id]; ok {
			delete(c.subs, id)
			close(cur)
		}
		c.mu.Unlock()
	}
}
