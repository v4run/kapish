package capi

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// clusterGVR is the GroupVersionResource for cluster.x-k8s.io/v1beta2 Clusters.
var clusterGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta2",
	Resource: "clusters",
}

// ListClusters returns all CAPI Clusters in the configured namespace (or all
// namespaces if Client.namespace == "").
func (c *Client) ListClusters(ctx context.Context) ([]Cluster, error) {
	res := c.dyn.Resource(clusterGVR)

	if c.namespace == "" {
		l, err := res.List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("capi: list clusters: %w", err)
		}
		out := make([]Cluster, 0, len(l.Items))
		for i := range l.Items {
			cl := &clusterv1.Cluster{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(l.Items[i].UnstructuredContent(), cl); err != nil {
				return nil, fmt.Errorf("capi: convert cluster %s/%s: %w", l.Items[i].GetNamespace(), l.Items[i].GetName(), err)
			}
			out = append(out, FromV1Beta2(cl))
		}
		return out, nil
	}

	l, err := res.Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("capi: list clusters in %s: %w", c.namespace, err)
	}
	out := make([]Cluster, 0, len(l.Items))
	for i := range l.Items {
		cl := &clusterv1.Cluster{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(l.Items[i].UnstructuredContent(), cl); err != nil {
			return nil, fmt.Errorf("capi: convert cluster %s/%s: %w", l.Items[i].GetNamespace(), l.Items[i].GetName(), err)
		}
		out = append(out, FromV1Beta2(cl))
	}
	return out, nil
}
