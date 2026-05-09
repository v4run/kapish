package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile_ValidBasic(t *testing.T) {
	c, err := LoadFromFile("testdata/valid_basic.yaml")
	require.NoError(t, err)

	assert.Equal(t, "prod-mgmt", c.ManagementClusters.Current)
	require.Len(t, c.ManagementClusters.Entries, 1)
	assert.Equal(t, "prod-mgmt", c.ManagementClusters.Entries[0].Name)
	assert.Equal(t, "/home/me/.kube/prod", c.ManagementClusters.Entries[0].Kubeconfig)

	assert.Equal(t, "/bin/zsh", c.Shell.Command)
	assert.Equal(t, "/tmp", c.Shell.Cwd)
	assert.Equal(t, "vim", c.Shell.Env["EDITOR"])
	assert.Equal(t, "kubectl", c.Shell.Aliases["k"])
	assert.Equal(t, "[{cluster}] $ ", c.Shell.Prompt)

	assert.Equal(t, "light", c.UI.Theme)
	assert.Equal(t, 60, c.UI.RefreshIntervalSec)
	assert.True(t, c.UI.OneShot)

	assert.Equal(t, 8080, c.Web.DefaultPort)
	assert.False(t, c.Web.OpenBrowser)
}

func TestLoadFromFile_MissingFileReturnsDefaults(t *testing.T) {
	c, err := LoadFromFile("testdata/does-not-exist.yaml")
	require.NoError(t, err, "missing file is not an error; defaults are returned")
	d := Defaults()
	assert.Equal(t, d, c)
}

func TestLoadFromFile_SyntaxErrorReturnsError(t *testing.T) {
	_, err := LoadFromFile("testdata/invalid_syntax.yaml")
	require.Error(t, err)
}

// LoadFromFile must overlay file values on top of defaults: keys absent in
// the file keep their default values.
func TestLoadFromFile_OverlaysDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`ui:
  theme: light
`), 0o600))

	c, err := LoadFromFile(path)
	require.NoError(t, err)

	d := Defaults()
	assert.Equal(t, "light", c.UI.Theme, "file value applied")
	assert.Equal(t, d.UI.RefreshIntervalSec, c.UI.RefreshIntervalSec, "absent keys use defaults")
	assert.Equal(t, d.Web.BindAddr, c.Web.BindAddr)
	assert.Equal(t, d.Shell.Prompt, c.Shell.Prompt)
}
