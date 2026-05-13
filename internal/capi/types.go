// Package capi wraps Cluster API types and exposes a small, focused API
// for kapish: list/watch CAPI Cluster CRDs and fetch workload kubeconfigs.
package capi

import (
	"strings"
	"time"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// Cluster is kapish's view of a CAPI Cluster. We don't expose the full
// v1beta2 type to consumers — TUI / Web only need a stable subset.
type Cluster struct {
	Name      string
	Namespace string

	Phase string

	ControlPlaneReady   bool
	InfrastructureReady bool

	K8sVersion string
	Provider   string

	CreationTimestamp time.Time
}

// FromV1Beta2 converts a CAPI v1beta2.Cluster into kapish's Cluster.
func FromV1Beta2(v *clusterv1.Cluster) Cluster {
	c := Cluster{
		Name:                v.Name,
		Namespace:           v.Namespace,
		Phase:               v.Status.Phase,
		ControlPlaneReady:   derefBool(v.Status.Initialization.ControlPlaneInitialized),
		InfrastructureReady: derefBool(v.Status.Initialization.InfrastructureProvisioned),
		CreationTimestamp:   v.CreationTimestamp.Time,
	}
	if v.Spec.Topology.Version != "" {
		c.K8sVersion = v.Spec.Topology.Version
	}
	if v.Spec.InfrastructureRef.Kind != "" {
		c.Provider = providerFromKind(v.Spec.InfrastructureRef.Kind)
	}
	return c
}

func derefBool(p *bool) bool { return p != nil && *p }

// providerFromKind extracts a short provider tag from the InfrastructureRef
// kind. Convention is <Provider>Cluster (AWSCluster, etc.). Unknown shapes
// return "".
func providerFromKind(kind string) string {
	if kind == "" {
		return ""
	}
	const suffix = "Cluster"
	if !strings.HasSuffix(kind, suffix) {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(kind, suffix))
}
