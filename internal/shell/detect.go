package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Kind is the v1 supported shell flavor.
type Kind int

const (
	KindUnknown Kind = iota
	KindBash
	KindZsh
	KindFish
)

func (k Kind) String() string {
	switch k {
	case KindBash:
		return "bash"
	case KindZsh:
		return "zsh"
	case KindFish:
		return "fish"
	}
	return "unknown"
}

// Detected is the result of resolving a shell.
type Detected struct {
	Path string
	Kind Kind
}

// Detect resolves the shell. Order of preference:
//  1. provided is non-empty (kapish config or override)
//  2. $SHELL env var
func Detect(provided string) (Detected, error) {
	path := provided
	if path == "" {
		path = os.Getenv("SHELL")
	}
	if path == "" {
		return Detected{}, errors.New("shell: no shell provided and $SHELL is unset")
	}
	// Bare names (e.g. "bash" from the settings UI) get resolved via $PATH;
	// anything with a slash is taken as a literal path and stat-ed.
	if !strings.ContainsRune(path, filepath.Separator) {
		resolved, err := exec.LookPath(path)
		if err != nil {
			return Detected{}, fmt.Errorf("shell: %s not found in $PATH: %w", path, err)
		}
		path = resolved
	}
	if _, err := os.Stat(path); err != nil {
		return Detected{}, fmt.Errorf("shell: %s not found: %w", path, err)
	}
	switch filepath.Base(path) {
	case "bash":
		return Detected{Path: path, Kind: KindBash}, nil
	case "zsh":
		return Detected{Path: path, Kind: KindZsh}, nil
	case "fish":
		return Detected{Path: path, Kind: KindFish}, nil
	default:
		return Detected{}, fmt.Errorf("shell: unsupported shell %q (v1: bash, zsh, fish)", filepath.Base(path))
	}
}
