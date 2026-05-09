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

func TestConfigEdit_RunsEditorAndValidates(t *testing.T) {
	// Use a tiny shell script as the "editor" — when invoked it overwrites
	// the file with valid YAML. We stage a target file the editor will modify.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: dark\n"), 0o600))

	editor := filepath.Join(dir, "editor.sh")
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\n"+
		"cat > \"$1\" <<'EOF'\n"+
		"ui:\n  theme: light\nEOF\n"), 0o755))

	t.Setenv("EDITOR", editor)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "edit", "--config", path})
	require.NoError(t, root.Execute())

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "theme: light")
}

func TestConfigEdit_RejectsInvalidEditedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: dark\n"), 0o600))

	editor := filepath.Join(dir, "editor.sh")
	// Editor introduces an unknown prompt token to trigger validation failure.
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\n"+
		"cat > \"$1\" <<'EOF'\n"+
		"shell:\n  prompt: \"{not_a_token}\"\nEOF\n"), 0o755))

	t.Setenv("EDITOR", editor)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "edit", "--config", path})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "{not_a_token}")
}

func TestConfigEdit_CreatesFromTemplateWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kapish", "config.yaml")
	// Editor that no-ops, leaving the template intact.
	editor := filepath.Join(dir, "editor.sh")
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	t.Setenv("EDITOR", editor)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "edit", "--config", path})
	require.NoError(t, root.Execute())

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "# kapish config")
}
