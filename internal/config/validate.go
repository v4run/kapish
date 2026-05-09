package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// supportedShells is the v1 allowlist of basenames. v2 may add nu, pwsh.
var supportedShells = map[string]bool{
	"bash": true,
	"zsh":  true,
	"fish": true,
}

func validateShell(s ShellConfig) []error {
	var errs []error
	if s.Command == "" {
		// empty -> resolved to $SHELL at spawn time; deferred check
		return errs
	}
	base := filepath.Base(s.Command)
	if !supportedShells[base] {
		errs = append(errs, fmt.Errorf("config: unsupported shell %q (v1 supports bash, zsh, fish)", base))
	}
	if _, err := exec.LookPath(s.Command); err != nil {
		// LookPath fails on absolute paths if the file isn't there; cross-check os.Stat.
		if _, statErr := os.Stat(s.Command); errors.Is(statErr, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("config: shell binary %q not found", s.Command))
		}
	}
	return errs
}
