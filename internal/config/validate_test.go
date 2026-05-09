package config

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateShell_EmptyCommandOK(t *testing.T) {
	// Empty command means "use $SHELL at spawn time" — accepted.
	errs := validateShell(ShellConfig{Command: ""})
	assert.Empty(t, errs)
}

func TestValidateShell_KnownShellInPath(t *testing.T) {
	// Pick a shell that is virtually always installed.
	bash, err := exec.LookPath("bash")
	require.NoError(t, err, "this test assumes bash is on PATH")

	errs := validateShell(ShellConfig{Command: bash})
	assert.Empty(t, errs)
}

func TestValidateShell_UnsupportedBasename(t *testing.T) {
	// /bin/ksh has a basename `ksh` which v1 doesn't support.
	errs := validateShell(ShellConfig{Command: "/usr/bin/ksh"})
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "unsupported shell")
}

func TestValidateShell_NotInPath(t *testing.T) {
	errs := validateShell(ShellConfig{Command: "/totally/not/a/real/zsh"})
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "not found")
}
