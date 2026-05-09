package config

import (
	"errors"
	"path/filepath"
)

// PathSources captures the inputs to ResolvePath. Caller fills in whichever
// it has. Empty fields are skipped.
type PathSources struct {
	Flag          string // --config flag value
	EnvVar        string // $KAPISH_CONFIG
	XDGConfigHome string // $XDG_CONFIG_HOME
	Home          string // $HOME (fallback when XDG unset)
}

// ResolvePath returns the path that kapish will read/write.
// Priority: Flag > EnvVar > XDGConfigHome/kapish/config.yaml > Home/.config/kapish/config.yaml.
// Returns an error if none of the sources is populated.
func ResolvePath(s PathSources) (string, error) {
	if s.Flag != "" {
		return s.Flag, nil
	}
	if s.EnvVar != "" {
		return s.EnvVar, nil
	}
	if s.XDGConfigHome != "" {
		return filepath.Join(s.XDGConfigHome, "kapish", "config.yaml"), nil
	}
	if s.Home != "" {
		return filepath.Join(s.Home, ".config", "kapish", "config.yaml"), nil
	}
	return "", errors.New("config: no flag, env, XDG, or HOME source available to resolve path")
}
