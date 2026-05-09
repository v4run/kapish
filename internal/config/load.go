package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFromFile reads the YAML file at path and overlays it on Defaults().
// If the file does not exist, Defaults() is returned with no error
// (kapish's first run boots with defaults).
// Syntax errors and other I/O errors are surfaced.
func LoadFromFile(path string) (Config, error) {
	c := Defaults()
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	// yaml.Unmarshal won't initialize nil maps if the file omits them,
	// but the Defaults() preamble already set non-nil empty maps. Some
	// edge cases (file says `shell:` with nothing under it) zero them
	// out in v3 — re-establish.
	if c.Shell.Env == nil {
		c.Shell.Env = map[string]string{}
	}
	if c.Shell.Aliases == nil {
		c.Shell.Aliases = map[string]string{}
	}
	return c, nil
}
