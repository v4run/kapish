package capi

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FetchKubeconfig reads the workload cluster's kubeconfig from the Secret
// named "<clusterName>-kubeconfig" in the given namespace, key "value".
// This is the CAPI convention.
func (c *Client) FetchKubeconfig(ctx context.Context, namespace, clusterName string) ([]byte, error) {
	secretName := clusterName + "-kubeconfig"
	s, err := c.kube.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("capi: get secret %s/%s: %w", namespace, secretName, err)
	}
	v, ok := s.Data["value"]
	if !ok {
		return nil, fmt.Errorf("capi: secret %s/%s has no 'value' key", namespace, secretName)
	}
	return v, nil
}
