package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

// allowedPromptTokens are the v1 substitutable tokens. v2 may add {git}, {region}.
var allowedPromptTokens = map[string]bool{
	"cluster":  true,
	"ns":       true,
	"provider": true,
	"ctx":      true,
	"time":     true,
}

// promptTokenRE matches '{name}'. We use the regex to find tokens; we also
// independently check that every '{' has a matching '}' to catch
// "hello {cluster " kind of mistakes.
var promptTokenRE = regexp.MustCompile(`\{([A-Za-z_]+)\}`)

func validatePrompt(p string) []error {
	var errs []error
	if open, close := strings.Count(p, "{"), strings.Count(p, "}"); open != close {
		errs = append(errs, fmt.Errorf("config: prompt template has unbalanced { } braces"))
		return errs
	}
	for _, m := range promptTokenRE.FindAllStringSubmatch(p, -1) {
		if !allowedPromptTokens[m[1]] {
			errs = append(errs, fmt.Errorf("config: unknown prompt token {%s}", m[1]))
		}
	}
	return errs
}

func validateMgmt(m ManagementClustersConfig) []error {
	var errs []error
	seen := make(map[string]bool, len(m.Entries))
	for i, e := range m.Entries {
		if e.Name == "" {
			errs = append(errs, fmt.Errorf("config: managementClusters.entries[%d].name must not be empty", i))
			continue
		}
		if seen[e.Name] {
			errs = append(errs, fmt.Errorf("config: duplicate managementClusters.entries name %q", e.Name))
			continue
		}
		seen[e.Name] = true
	}
	if m.Current != "" && !seen[m.Current] {
		errs = append(errs, fmt.Errorf("config: managementClusters.current %q does not match any entry", m.Current))
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
