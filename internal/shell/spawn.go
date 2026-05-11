package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SpawnPlan is everything needed to launch a kapish shell. Caller wires
// stdio (or wraps in PTY) and calls Start/Run on Cmd. Cleanup MUST be
// called (defer) to remove the temp dir.
type SpawnPlan struct {
	Cmd        *exec.Cmd
	SessionDir *SessionDir
}

func (p *SpawnPlan) Cleanup() error {
	if p.SessionDir == nil {
		return nil
	}
	return p.SessionDir.Remove()
}

// PrepareSpawn detects the shell, creates a session dir, writes init files,
// and returns an unstarted exec.Cmd configured for that shell.
func PrepareSpawn(opts Options, kubeconfig []byte) (*SpawnPlan, error) {
	d, err := Detect(opts.PathToShell)
	if err != nil {
		return nil, err
	}

	sd, err := newSessionDir(kubeconfig)
	if err != nil {
		return nil, err
	}

	env := append([]string(nil), os.Environ()...)

	var cmd *exec.Cmd
	switch d.Kind {
	case KindBash:
		rcfile := filepath.Join(sd.Path, ".kapishrc")
		if err := os.WriteFile(rcfile, []byte(bashInit(opts, sd.KubeconfigPath)), 0o600); err != nil {
			_ = sd.Remove()
			return nil, fmt.Errorf("shell: write bash rcfile: %w", err)
		}
		cmd = exec.Command(d.Path, "--rcfile", rcfile)

	case KindZsh:
		zshrc := filepath.Join(sd.Path, ".zshrc")
		if err := os.WriteFile(zshrc, []byte(zshInit(opts, sd.KubeconfigPath)), 0o600); err != nil {
			_ = sd.Remove()
			return nil, fmt.Errorf("shell: write zshrc: %w", err)
		}
		env = append(env, "ZDOTDIR="+sd.Path)
		cmd = exec.Command(d.Path)

	case KindFish:
		init := fishInit(opts, sd.KubeconfigPath)
		cmd = exec.Command(d.Path, "--init-command="+init)

	default:
		_ = sd.Remove()
		return nil, fmt.Errorf("shell: unsupported kind %s", d.Kind)
	}

	cmd.Env = env

	return &SpawnPlan{Cmd: cmd, SessionDir: sd}, nil
}
