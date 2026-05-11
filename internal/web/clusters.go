package web

import (
	"net/http"
	"time"

	"github.com/v4run/kapish/internal/capi"
)

// clusterDTO is the JSON shape returned to the browser.
type clusterDTO struct {
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	Phase               string `json:"phase"`
	ControlPlaneReady   bool   `json:"controlPlaneReady"`
	InfrastructureReady bool   `json:"infrastructureReady"`
	Version             string `json:"version"`
	Provider            string `json:"provider"`
	AgeSeconds          int64  `json:"ageSeconds"`
}

func toDTO(c capi.Cluster) clusterDTO {
	d := clusterDTO{
		Name:                c.Name,
		Namespace:           c.Namespace,
		Phase:               c.Phase,
		ControlPlaneReady:   c.ControlPlaneReady,
		InfrastructureReady: c.InfrastructureReady,
		Version:             c.K8sVersion,
		Provider:            c.Provider,
	}
	if !c.CreationTimestamp.IsZero() {
		d.AgeSeconds = int64(time.Since(c.CreationTimestamp).Seconds())
	}
	return d
}

func (s *Server) handleGetClusters(w http.ResponseWriter, r *http.Request) {
	snap := s.cache.snapshot()
	out := make([]clusterDTO, 0, len(snap))
	for _, c := range snap {
		out = append(out, toDTO(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"clusters": out})
}
