package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const inputWithComments = `# kapish config
# user comment that must survive a write

managementClusters:
  current: prod
  entries:
    - name: prod              # the prod mgmt cluster
      kubeconfig: /home/me/.kube/prod
      context: prod-admin

shell:
  command: /bin/zsh
  prompt: "[{cluster}] $ "    # template with cluster token
  env:
    EDITOR: vim               # I prefer vim
  aliases:
    k: kubectl

ui:
  theme: dark
  refreshIntervalSec: 30
  oneShot: false

web:
  defaultPort: 0
  openBrowser: true
  bindAddr: 127.0.0.1
`

func TestWriteToFile_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(inputWithComments), 0o600))

	// Load the config, change the theme, write it back.
	c, err := LoadFromFile(path)
	require.NoError(t, err)
	c.UI.Theme = "light"

	require.NoError(t, WriteToFile(path, c))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	got := string(out)

	// Comments preserved
	assert.Contains(t, got, "# kapish config")
	assert.Contains(t, got, "# user comment that must survive a write")
	assert.Contains(t, got, "# the prod mgmt cluster")
	assert.Contains(t, got, "# template with cluster token")
	assert.Contains(t, got, "# I prefer vim")

	// Theme value updated
	// We assert this via a second load round-trip to avoid coupling
	// the test to exact whitespace.
	c2, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c2.UI.Theme)
}

func TestWriteToFile_CreatesParentDirAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "kapish", "config.yaml")
	c := Defaults()
	c.UI.Theme = "light"

	require.NoError(t, WriteToFile(path, c))

	c2, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c2.UI.Theme)

	// Permissions: file 0600, parent dir 0700.
	fi, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	pi, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), pi.Mode().Perm())
}

func TestWriteToFile_NoExistingFile_StillWritesValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	c := Defaults()
	c.Shell.Env = map[string]string{"EDITOR": "vim"}

	require.NoError(t, WriteToFile(path, c))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	// produced YAML round-trips
	c2, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "vim", c2.Shell.Env["EDITOR"])
	assert.True(t, strings.Contains(string(got), "EDITOR"))
}
