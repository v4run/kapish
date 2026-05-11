package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_OptionsTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "zsh")
	require.NoError(t, os.WriteFile(fakeShell, []byte("#!/bin/sh\n"), 0o755))

	d, err := Detect(fakeShell)
	require.NoError(t, err)
	assert.Equal(t, fakeShell, d.Path)
	assert.Equal(t, KindZsh, d.Kind)
}

func TestDetect_FallsBackToShellEnv(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "bash")
	require.NoError(t, os.WriteFile(fakeShell, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("SHELL", fakeShell)

	d, err := Detect("")
	require.NoError(t, err)
	assert.Equal(t, fakeShell, d.Path)
	assert.Equal(t, KindBash, d.Kind)
}

func TestDetect_UnknownBasename(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "ksh")
	require.NoError(t, os.WriteFile(fakeShell, []byte("#!/bin/sh\n"), 0o755))

	_, err := Detect(fakeShell)
	require.Error(t, err)
}

func TestDetect_NotInPath(t *testing.T) {
	_, err := Detect("/totally/not/here/zsh")
	require.Error(t, err)
}
