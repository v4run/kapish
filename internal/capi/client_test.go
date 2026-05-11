package capi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tinyKubeconfig = `apiVersion: v1
kind: Config
current-context: test-ctx
clusters:
- name: c1
  cluster:
    server: https://localhost:6443
contexts:
- name: test-ctx
  context:
    cluster: c1
    user: test-user
- name: alt-ctx
  context:
    cluster: c1
    user: test-user
users:
- name: test-user
  user:
    token: redacted
`

func TestNewClient_LoadsKubeconfigCurrentContext(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(tinyKubeconfig), 0o600))

	c, err := NewClient(Options{Kubeconfig: kubeconfig})
	require.NoError(t, err)
	assert.Equal(t, "test-ctx", c.Context())
}

func TestNewClient_OverrideContext(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(tinyKubeconfig), 0o600))

	c, err := NewClient(Options{Kubeconfig: kubeconfig, Context: "alt-ctx"})
	require.NoError(t, err)
	assert.Equal(t, "alt-ctx", c.Context())
}

func TestNewClient_MissingKubeconfig(t *testing.T) {
	_, err := NewClient(Options{Kubeconfig: "/totally/nope/kubeconfig"})
	require.Error(t, err)
}
