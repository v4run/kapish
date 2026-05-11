package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionDir_CreatesAndWritesKubeconfig(t *testing.T) {
	d, err := newSessionDir([]byte("# kubeconfig content\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Remove() })

	fi, err := os.Stat(d.Path)
	require.NoError(t, err)
	assert.True(t, fi.IsDir())
	assert.Equal(t, os.FileMode(0o700), fi.Mode().Perm())

	got, err := os.ReadFile(d.KubeconfigPath)
	require.NoError(t, err)
	assert.Equal(t, "# kubeconfig content\n", string(got))

	kfi, err := os.Stat(d.KubeconfigPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), kfi.Mode().Perm())

	rel, err := filepath.Rel(os.TempDir(), d.Path)
	require.NoError(t, err)
	assert.False(t, filepath.IsAbs(rel))
	assert.Contains(t, filepath.Base(d.Path), "kapish-")
}

func TestSessionDir_RemoveIsIdempotent(t *testing.T) {
	d, err := newSessionDir(nil)
	require.NoError(t, err)
	require.NoError(t, d.Remove())
	require.NoError(t, d.Remove())
}
