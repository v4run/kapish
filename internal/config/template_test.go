package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureFirstRunTemplate_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kapish", "config.yaml")

	created, err := EnsureFirstRunTemplate(path)
	require.NoError(t, err)
	assert.True(t, created, "should report it created the file")

	got, err := os.ReadFile(path)
	require.NoError(t, err)

	// Has guiding comment header
	assert.Contains(t, string(got), "# kapish config")
	// Has every top-level section commented out so users see it
	assert.Contains(t, string(got), "managementClusters")
	assert.Contains(t, string(got), "shell")
	assert.Contains(t, string(got), "ui")
	assert.Contains(t, string(got), "web")

	// File parses as valid YAML and yields default values when loaded.
	c, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, Defaults(), c)
}

func TestEnsureFirstRunTemplate_NoOpWhenExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: light\n"), 0o600))

	created, err := EnsureFirstRunTemplate(path)
	require.NoError(t, err)
	assert.False(t, created)

	c, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c.UI.Theme, "existing file untouched")
}
