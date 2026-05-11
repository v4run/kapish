package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/v4run/kapish/internal/capi"
)

func (s *Server) handleClustersStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	sub, unsub := s.cache.subscribe()
	defer unsub()

	// Initial: tell the client to re-fetch the snapshot (simplest & robust).
	fmt.Fprintf(w, "event: sync\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			b, _ := json.Marshal(sseEventDTO(ev))
			fmt.Fprintf(w, "event: cluster\ndata: %s\n\n", b)
			flusher.Flush()
		}
	}
}

type sseEvent struct {
	Type    string     `json:"type"` // "added" | "modified" | "deleted"
	Cluster clusterDTO `json:"cluster"`
}

func sseEventDTO(ev capi.Event) sseEvent {
	t := "modified"
	switch ev.Type {
	case capi.EventAdded:
		t = "added"
	case capi.EventDeleted:
		t = "deleted"
	}
	return sseEvent{Type: t, Cluster: toDTO(ev.Cluster)}
}
