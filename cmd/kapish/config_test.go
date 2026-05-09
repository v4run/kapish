package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate_PrintsEffectiveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`ui:
  theme: light
`), 0o600))

	root := newRootCmd()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"config", "validate", "--config", path})
	require.NoError(t, root.Execute())

	got := stdout.String()
	// Effective config should contain the override.
	assert.Contains(t, got, "theme: light")
	// And the default for an unset value.
	assert.Contains(t, got, "refreshIntervalSec: 30")
}

func TestConfigValidate_FailsOnInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// alias name with space violates the alias regex.
	require.NoError(t, os.WriteFile(path, []byte(`shell:
  aliases:
    "bad alias": kubectl
`), 0o600))

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "validate", "--config", path})
	err := root.Execute()
	require.Error(t, err)
	assert.True(t, strings.Contains(buf.String(), "alias") || strings.Contains(err.Error(), "alias"))
}
