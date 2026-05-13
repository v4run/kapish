package capi

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// EventType classifies a cluster watch event.
type EventType int

const (
	EventAdded EventType = iota
	EventModified
	EventDeleted
	EventError
)

// Event is one cluster lifecycle event.
type Event struct {
	Type    EventType
	Cluster Cluster
	Err     error
}

// WatchClusters returns a channel of Cluster events. The channel is closed
// when ctx is canceled or the underlying watcher shuts down. Caller is
// responsible for context lifecycle and any backoff/reconnect logic.
func (c *Client) WatchClusters(ctx context.Context) (<-chan Event, error) {
	li := c.dyn.Resource(clusterGVR)
	var w watch.Interface
	var err error
	if c.namespace == "" {
		w, err = li.Watch(ctx, metav1.ListOptions{})
	} else {
		w, err = li.Namespace(c.namespace).Watch(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("capi: watch clusters: %w", err)
	}

	out := make(chan Event, 16)
	go func() {
		defer close(out)
		defer w.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.ResultChan():
				if !ok {
					return
				}
				kev := translateWatchEvent(ev)
				select {
				case <-ctx.Done():
					return
				case out <- kev:
				}
			}
		}
	}()
	return out, nil
}

func translateWatchEvent(ev watch.Event) Event {
	switch ev.Type {
	case watch.Added:
		return Event{Type: EventAdded, Cluster: clusterFromObject(ev.Object)}
	case watch.Modified:
		return Event{Type: EventModified, Cluster: clusterFromObject(ev.Object)}
	case watch.Deleted:
		return Event{Type: EventDeleted, Cluster: clusterFromObject(ev.Object)}
	case watch.Error:
		return Event{Type: EventError, Err: fmt.Errorf("capi watch error: %v", ev.Object)}
	}
	return Event{Type: EventError, Err: fmt.Errorf("capi: unknown watch event type %q", ev.Type)}
}

func clusterFromObject(obj runtime.Object) Cluster {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return Cluster{}
	}
	cl := &clusterv1.Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), cl); err != nil {
		return Cluster{}
	}
	return FromV1Beta2(cl)
}
