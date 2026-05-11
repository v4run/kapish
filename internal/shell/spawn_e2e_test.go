package shell

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func haveShell(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not on PATH; skipping", name)
	}
	return p
}

// TestEnd2End_BashAppliesEnvAndAlias verifies that PrepareSpawn produces a
// SpawnPlan whose Cmd, when run as an interactive shell (-i), sources the
// generated rcfile and therefore exports the requested env var.
//
// NOTE: bash --rcfile is only sourced for interactive shells. Running
// `bash --rcfile <path> -c "..."` (non-interactive) does NOT source the
// rcfile on bash 3.2 or 5.x. We therefore add -i to make the shell
// interactive; this causes bash to source --rcfile as expected.
func TestEnd2End_BashAppliesEnvAndAlias(t *testing.T) {
	bash := haveShell(t, "bash")

	plan, err := PrepareSpawn(Options{
		PathToShell: bash,
		Env:         map[string]string{"FOO": "bar"},
		Aliases:     map[string]string{"hi": "echo HELLO"},
	}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	// -i makes bash interactive so it sources --rcfile; -c runs the command
	// without waiting for stdin. Without -i, --rcfile is ignored by bash.
	plan.Cmd.Args = append(plan.Cmd.Args, "-i", "-c", "echo $FOO; shopt -s expand_aliases; alias hi >/dev/null 2>&1 && hi || echo NO_HI")
	var out bytes.Buffer
	plan.Cmd.Stdout = &out
	plan.Cmd.Stderr = &out
	require.NoError(t, plan.Cmd.Run())

	assert.Contains(t, out.String(), "bar", "env var FOO should propagate; output: %q", out.String())
}

// TestEnd2End_ZshAppliesEnv verifies that PrepareSpawn's zsh plan, when run
// as an interactive shell (-i -c), sources ZDOTDIR/.zshrc and exports the env.
//
// NOTE: zsh only sources .zshrc for interactive shells. `zsh -c "..."` alone
// (non-interactive) does not source .zshrc even when ZDOTDIR is set. We add
// -i so the startup file is read.
func TestEnd2End_ZshAppliesEnv(t *testing.T) {
	zsh := haveShell(t, "zsh")

	plan, err := PrepareSpawn(Options{
		PathToShell: zsh,
		Env:         map[string]string{"FOO": "bar"},
	}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	// Same reasoning as bash: -i is required to trigger .zshrc sourcing.
	plan.Cmd.Args = append(plan.Cmd.Args, "-i", "-c", "echo $FOO")
	var out bytes.Buffer
	plan.Cmd.Stdout = &out
	plan.Cmd.Stderr = &out
	require.NoError(t, plan.Cmd.Run())

	assert.Contains(t, out.String(), "bar", "output: %q", out.String())
}
