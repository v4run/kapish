package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// supportedShells is the v1 allowlist of basenames. v2 may add nu, pwsh.
var supportedShells = map[string]bool{
	"bash": true,
	"zsh":  true,
	"fish": true,
}

// envKeyRE matches POSIX-style env var names.
var envKeyRE = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// aliasNameRE allows shell identifier characters used in practice. Aliases
// must not contain whitespace or '='. We allow letters, digits, underscore,
// and hyphen; cannot start with a digit.
var aliasNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_\-]*$`)

func validateEnv(env map[string]string) []error {
	var errs []error
	for k := range env {
		if !envKeyRE.MatchString(k) {
			errs = append(errs, fmt.Errorf("config: env key %q must match [A-Z_][A-Z0-9_]*", k))
		}
	}
	return errs
}

func validateAliases(a map[string]string) []error {
	var errs []error
	for name := range a {
		if !aliasNameRE.MatchString(name) {
			errs = append(errs, fmt.Errorf("config: alias name %q is not a valid identifier", name))
		}
	}
	return errs
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
