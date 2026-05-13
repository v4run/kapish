package capi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

func TestFromV1Beta2_PopulatesFields(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "prod-eu-1",
			Namespace:         "prod",
			CreationTimestamp: metav1.NewTime(created),
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{Kind: "AWSCluster"},
			Topology:          clusterv1.Topology{Version: "v1.30.2"},
		},
		Status: clusterv1.ClusterStatus{
			Phase: "Provisioned",
			Initialization: clusterv1.ClusterInitializationStatus{
				ControlPlaneInitialized:   ptr.To(true),
				InfrastructureProvisioned: ptr.To(true),
			},
		},
	}

	c := FromV1Beta2(v1)
	assert.Equal(t, "prod-eu-1", c.Name)
	assert.Equal(t, "prod", c.Namespace)
	assert.Equal(t, "Provisioned", c.Phase)
	assert.True(t, c.ControlPlaneReady)
	assert.True(t, c.InfrastructureReady)
	assert.Equal(t, "v1.30.2", c.K8sVersion)
	assert.Equal(t, "aws", c.Provider)
	assert.Equal(t, created, c.CreationTimestamp)
}

func TestFromV1Beta2_EmptyInfrastructureRefAndNilInit(t *testing.T) {
	v1 := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "no-infra"},
		Status:     clusterv1.ClusterStatus{Phase: "Pending"},
	}
	c := FromV1Beta2(v1)
	assert.Equal(t, "", c.Provider)
	assert.Equal(t, "", c.K8sVersion)
	assert.False(t, c.ControlPlaneReady)
	assert.False(t, c.InfrastructureReady)
}

func TestProviderFromKind(t *testing.T) {
	cases := map[string]string{
		"AWSCluster":       "aws",
		"GCPCluster":       "gcp",
		"AzureCluster":     "azure",
		"VSphereCluster":   "vsphere",
		"HetznerCluster":   "hetzner",
		"OpenStackCluster": "openstack",
		"":                 "",
	}
	for kind, want := range cases {
		require.Equal(t, want, providerFromKind(kind), "kind=%s", kind)
	}
}
