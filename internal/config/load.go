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

// FlagOverrides captures values from CLI flags that override file/defaults.
// Pointer fields differentiate "not set" from "explicitly false/empty".
type FlagOverrides struct {
	Kubeconfig string // --kubeconfig
	Context    string // --context
	OneShot    *bool  // --one-shot
}

// ApplyOverrides returns a copy of c with overrides applied. Order of effect
// is already encoded by the call sites (defaults < file < env < flags), so
// this function only handles the final flag layer.
//
// Mgmt-cluster overrides target the entry referenced by ManagementClusters.Current.
// If no current is set and no entries exist, a synthetic "default" entry is
// created so downstream consumers always have something to use.
func ApplyOverrides(c Config, o FlagOverrides) Config {
	out := c

	if o.Kubeconfig != "" || o.Context != "" {
		if len(out.ManagementClusters.Entries) == 0 {
			out.ManagementClusters.Entries = []ManagementClusterEntry{{Name: "default"}}
			out.ManagementClusters.Current = "default"
		} else {
			// Copy the slice so mutations don't bleed into the caller's c.
			copied := make([]ManagementClusterEntry, len(out.ManagementClusters.Entries))
			copy(copied, out.ManagementClusters.Entries)
			out.ManagementClusters.Entries = copied
		}
		idx := indexOfCurrent(out.ManagementClusters)
		if idx < 0 {
			idx = 0
			out.ManagementClusters.Current = out.ManagementClusters.Entries[0].Name
		}
		if o.Kubeconfig != "" {
			out.ManagementClusters.Entries[idx].Kubeconfig = o.Kubeconfig
		}
		if o.Context != "" {
			out.ManagementClusters.Entries[idx].Context = o.Context
		}
	}

	if o.OneShot != nil {
		out.UI.OneShot = *o.OneShot
	}

	return out
}

func indexOfCurrent(m ManagementClustersConfig) int {
	if m.Current == "" {
		return -1
	}
	for i, e := range m.Entries {
		if e.Name == m.Current {
			return i
		}
	}
	return -1
}
