package shell

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SessionDir is a per-spawn temp dir holding the kubeconfig and any
// shell init files. Caller MUST call Remove() (use defer).
type SessionDir struct {
	Path           string
	KubeconfigPath string
	removed        bool
}

func newSessionDir(kubeconfig []byte) (*SessionDir, error) {
	dir, err := os.MkdirTemp("", "kapish-*")
	if err != nil {
		return nil, fmt.Errorf("shell: mkdtemp: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("shell: chmod %s: %w", dir, err)
	}
	kpath := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(kpath, kubeconfig, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("shell: write kubeconfig: %w", err)
	}
	return &SessionDir{Path: dir, KubeconfigPath: kpath}, nil
}

// Remove deletes the temp dir. Idempotent.
func (d *SessionDir) Remove() error {
	if d.removed {
		return nil
	}
	d.removed = true
	if err := os.RemoveAll(d.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("shell: cleanup %s: %w", d.Path, err)
	}
	return nil
}
