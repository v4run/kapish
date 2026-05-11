package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareSpawn_BashUsesRcfileFlag(t *testing.T) {
	dir := t.TempDir()
	bash := filepath.Join(dir, "bash")
	require.NoError(t, os.WriteFile(bash, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{
		PathToShell: bash,
		Env:         map[string]string{"FOO": "bar"},
	}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	require.NotNil(t, plan.Cmd)
	assert.Equal(t, bash, plan.Cmd.Path)
	args := plan.Cmd.Args
	require.True(t, len(args) >= 3, "expected --rcfile + path: %v", args)
	rcIdx := -1
	for i, a := range args {
		if a == "--rcfile" {
			rcIdx = i
		}
	}
	require.True(t, rcIdx >= 0 && rcIdx+1 < len(args))
	rcfile := args[rcIdx+1]
	body, err := os.ReadFile(rcfile)
	require.NoError(t, err)
	assert.Contains(t, string(body), "export FOO='bar'")
	assert.Contains(t, string(body), "export KUBECONFIG=")
}

func TestPrepareSpawn_ZshSetsZDOTDIR(t *testing.T) {
	dir := t.TempDir()
	zsh := filepath.Join(dir, "zsh")
	require.NoError(t, os.WriteFile(zsh, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{PathToShell: zsh}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	var zdot string
	for _, e := range plan.Cmd.Env {
		if strings.HasPrefix(e, "ZDOTDIR=") {
			zdot = strings.TrimPrefix(e, "ZDOTDIR=")
		}
	}
	require.NotEmpty(t, zdot)
	body, err := os.ReadFile(filepath.Join(zdot, ".zshrc"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "export KUBECONFIG=")
}

func TestPrepareSpawn_FishUsesInitCommand(t *testing.T) {
	dir := t.TempDir()
	fish := filepath.Join(dir, "fish")
	require.NoError(t, os.WriteFile(fish, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{PathToShell: fish}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	hasInit := false
	for _, a := range plan.Cmd.Args {
		if strings.HasPrefix(a, "--init-command=") {
			hasInit = true
			assert.Contains(t, a, "set -gx KUBECONFIG")
		}
	}
	assert.True(t, hasInit)
}

func TestPrepareSpawn_CleanupRemovesDir(t *testing.T) {
	dir := t.TempDir()
	bash := filepath.Join(dir, "bash")
	require.NoError(t, os.WriteFile(bash, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{PathToShell: bash}, []byte("# kc\n"))
	require.NoError(t, err)

	sessionPath := plan.SessionDir.Path
	require.NoError(t, plan.Cleanup())
	_, err = os.Stat(sessionPath)
	assert.Error(t, err)
}
