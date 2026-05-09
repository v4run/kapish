package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// firstRunTemplate is the YAML written if no kapish config file exists.
// Every top-level section is present (commented or with default values)
// so users discover what's configurable without having to read docs.
const firstRunTemplate = `# kapish config — see docs/superpowers/specs/2026-05-09-kapish-design.md
#
# Resolution order: --config flag > $KAPISH_CONFIG > $XDG_CONFIG_HOME/kapish/config.yaml.
# Edit with:  kapish config edit
# Validate :  kapish config validate

# Management clusters that kapish queries for CAPI Cluster resources.
# 'current' must match an entry's name.
# managementClusters:
#   current: prod-mgmt
#   entries:
#     - name: prod-mgmt
#       kubeconfig: ""        # default: $KUBECONFIG or ~/.kube/config
#       context: ""           # default: kubeconfig's current-context
#       namespace: ""         # default: all namespaces

# Shell behavior for the spawned cluster shell.
shell:
  command: ""                 # default: $SHELL ('/bin/bash' fallback). v1: bash, zsh, fish.
  cwd: ""                     # working directory (empty = inherit)
  env: {}                     # injected env vars, e.g. { EDITOR: vim }
  aliases: {}                 # injected aliases, e.g. { k: kubectl, kgp: "kubectl get pods -A" }
  prompt: "[{cluster}] "      # tokens: {cluster} {ns} {provider} {ctx} {time}

ui:
  theme: dark                 # dark | light
  refreshIntervalSec: 30
  oneShot: false              # true: TUI exits after first shell exit

web:
  defaultPort: 0              # 0 = pick free port
  openBrowser: true
  bindAddr: "127.0.0.1"       # any non-loopback requires --bind override
`

// EnsureFirstRunTemplate writes the first-run template to path if no file is
// already there. Returns whether it created the file. If the parent directory
// is missing it is created with 0700.
func EnsureFirstRunTemplate(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("config: stat %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(firstRunTemplate), 0o600); err != nil {
		return false, fmt.Errorf("config: write %s: %w", path, err)
	}
	return true, nil
}
