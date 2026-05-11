package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/v4run/kapish/internal/capi"
)

func (s *Server) handleGetMgmts(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Name      string `json:"name"`
		Context   string `json:"context,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	}
	mc := s.opts.AppConfig.ManagementClusters
	entries := make([]entry, 0, len(mc.Entries))
	for _, e := range mc.Entries {
		entries = append(entries, entry{Name: e.Name, Context: e.Context, Namespace: e.Namespace})
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": mc.Current, "entries": entries})
}

func (s *Server) handlePutMgmtsCurrent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected {\"name\":\"...\"}"})
		return
	}
	mc := s.opts.AppConfig.ManagementClusters
	var picked *struct {
		Kubeconfig, Context, Namespace string
	}
	for _, e := range mc.Entries {
		if e.Name == body.Name {
			picked = &struct{ Kubeconfig, Context, Namespace string }{e.Kubeconfig, e.Context, e.Namespace}
			break
		}
	}
	if picked == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no such mgmt cluster: " + body.Name})
		return
	}

	// Rebuild the capi client if we have one (skip in test mode).
	if s.opts.CapiClient != nil {
		c, err := capi.NewClient(capi.Options{Kubeconfig: picked.Kubeconfig, Context: picked.Context, Namespace: picked.Namespace})
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cs, err := c.ListClusters(ctx)
		cancel()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		s.opts.CapiClient = c
		s.opts.MgmtContext = c.Context()
		s.cache.replaceAll(cs)
	}
	s.opts.AppConfig.ManagementClusters.Current = body.Name
	writeJSON(w, http.StatusOK, map[string]string{"status": "switched", "current": body.Name})
}
