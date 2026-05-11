package capi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestFetchKubeconfig_HappyPath(t *testing.T) {
	const want = `apiVersion: v1
kind: Config
contexts: []
`
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-eu-1-kubeconfig",
			Namespace: "prod",
		},
		Data: map[string][]byte{"value": []byte(want)},
	}
	kube := fake.NewSimpleClientset(secret)
	c := &Client{kube: kube}

	got, err := c.FetchKubeconfig(context.Background(), "prod", "prod-eu-1")
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestFetchKubeconfig_MissingSecret(t *testing.T) {
	kube := fake.NewSimpleClientset()
	c := &Client{kube: kube}

	_, err := c.FetchKubeconfig(context.Background(), "prod", "missing")
	require.Error(t, err)
}

func TestFetchKubeconfig_MissingValueKey(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "weird-kubeconfig",
			Namespace: "prod",
		},
		Data: map[string][]byte{"otherKey": []byte("hi")},
	}
	kube := fake.NewSimpleClientset(secret)
	c := &Client{kube: kube}

	_, err := c.FetchKubeconfig(context.Background(), "prod", "weird")
	require.Error(t, err)
}
