# Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the kapish project as a `go install`-able Go binary with cobra CLI, a fully tested config package (load → merge → validate → comment-preserving write), first-run template generation, structured logging, and the global flags that all later plans will rely on.

**Architecture:** Single Go module at `github.com/v4run/kapish`. Single binary entry point under `cmd/kapish/`. Internal packages live under `internal/` and are not part of the public API. Config is layered (built-in defaults → file → env → flags), validated against a typed schema, and written back via yaml.v3's `Node` API to preserve user comments and key ordering. Version info uses `runtime/debug.ReadBuildInfo()` so `go install` works without ldflags, with ldflags as an override path for `make build`.

**Tech Stack:**
- Go 1.22+ (stable `log/slog`)
- `github.com/spf13/cobra` v1.8+ — CLI
- `gopkg.in/yaml.v3` — YAML parse + Node-level write
- `github.com/gofrs/flock` v0.8+ — file locking on writes
- `gopkg.in/natefinch/lumberjack.v2` — log file rotation
- `github.com/stretchr/testify` v1.9+ — test assertions

**`go install` end-state for this plan:**
```bash
go install github.com/v4run/kapish/cmd/kapish@latest
kapish version          # prints version (BuildInfo-derived)
kapish config validate  # prints merged effective config
kapish config edit      # opens config in $EDITOR
```

---

## Task 1: Initialize Go module, git repo, and project skeleton

**Files:**
- Create: `/Users/varun/projects/personal/kapish/go.mod`
- Create: `/Users/varun/projects/personal/kapish/.gitignore`
- Create: `/Users/varun/projects/personal/kapish/README.md`
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/main.go`

- [ ] **Step 1: Initialize git repo**

Run: `git init`
Expected: `Initialized empty Git repository in /Users/varun/projects/personal/kapish/.git/`

- [ ] **Step 2: Initialize Go module**

Run: `go mod init github.com/v4run/kapish`
Expected: `go: creating new go.mod: module github.com/v4run/kapish`

- [ ] **Step 3: Write the .gitignore**

Create `.gitignore`:

```gitignore
# binaries
/bin/
/dist/

# Go test artifacts
*.test
*.out
coverage.txt

# editor / OS
.DS_Store
.idea/
.vscode/
*.swp

# kapish runtime
*.kubeconfig
kapish-*.tmp/
```

- [ ] **Step 4: Write a minimal README.md**

Create `README.md`:

````markdown
# kapish

A debugging tool for Cluster API (CAPI) — list workload clusters from a management cluster and drop into a shell whose `KUBECONFIG` is pre-set to the chosen cluster, with configurable env vars, aliases, working directory, and prompt prefix. Runs as a TUI or as a localhost web app.

## Install

```sh
go install github.com/v4run/kapish/cmd/kapish@latest
```

Make sure `$(go env GOBIN)` (or `$(go env GOPATH)/bin` if `GOBIN` is unset) is in your `PATH`.

## Quick start

```sh
kapish              # TUI
kapish serve        # web UI (opens browser)
kapish version
kapish config edit
```

## Status

In active development. See `docs/superpowers/specs/` for design and `docs/superpowers/plans/` for implementation plans.
````

- [ ] **Step 5: Write a minimal main.go that compiles**

Create `cmd/kapish/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("kapish dev")
		return
	}
	fmt.Fprintln(os.Stderr, "kapish: cobra wiring lands in Task 2")
	os.Exit(2)
}
```

- [ ] **Step 6: Verify the build**

Run: `go build ./cmd/kapish && ./kapish version`
Expected: `kapish dev`

- [ ] **Step 7: Verify `go install` works**

Run: `go install ./cmd/kapish && "$(go env GOPATH)/bin/kapish" version`
Expected: `kapish dev`

- [ ] **Step 8: Commit**

```bash
git add .gitignore README.md go.mod cmd/kapish/main.go
git commit -m "chore: initialize go module and skeleton"
```

---

## Task 2: Wire cobra root command

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/cmd/kapish/main.go`
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/root.go`

- [ ] **Step 1: Add cobra dependency**

Run: `go get github.com/spf13/cobra@latest`
Expected: `go: added github.com/spf13/cobra v1.x.x`

- [ ] **Step 2: Write root.go with the root cobra command**

Create `cmd/kapish/root.go`:

```go
package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage: true,
	}
	return root
}
```

- [ ] **Step 3: Replace main.go with cobra-driven entry point**

Replace `cmd/kapish/main.go` with:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kapish:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Build and verify usage prints**

Run: `go build ./cmd/kapish && ./kapish --help`
Expected: usage text containing `kapish — pick a CAPI cluster and drop into a shell`.

- [ ] **Step 5: Commit**

```bash
git add cmd/kapish/main.go cmd/kapish/root.go go.mod go.sum
git commit -m "feat(cli): add cobra root command"
```

---

## Task 3: Add `version` subcommand backed by BuildInfo

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/version/version.go`
- Create: `/Users/varun/projects/personal/kapish/internal/version/version_test.go`
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/version.go`

- [ ] **Step 1: Write the failing test**

Create `internal/version/version_test.go`:

```go
package version

import (
	"strings"
	"testing"
)

func TestStringFallsBackToDev(t *testing.T) {
	// In a unit-test build, BuildInfo's main module version is usually "(devel)"
	// and there's no VCS info, so String() should fall back to ldflags Version
	// (default "dev").
	got := String()
	if got == "" {
		t.Fatalf("String() returned empty")
	}
	// dev or a 7-char rev or a semver — all acceptable. We just verify it's not empty
	// and doesn't contain whitespace.
	if strings.ContainsAny(got, " \t\n") {
		t.Fatalf("String() = %q, must not contain whitespace", got)
	}
}

func TestLongIncludesString(t *testing.T) {
	long := Long()
	if !strings.Contains(long, String()) {
		t.Fatalf("Long() = %q, expected to contain String() = %q", long, String())
	}
	if !strings.HasPrefix(long, "kapish ") {
		t.Fatalf("Long() = %q, expected to start with 'kapish '", long)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/version -v`
Expected: FAIL — package `internal/version` does not exist yet.

- [ ] **Step 3: Implement version package**

Create `internal/version/version.go`:

```go
package version

import (
	"fmt"
	"runtime/debug"
)

// Set via -ldflags in Makefile builds; ignored under `go install`.
var (
	Version = "dev"
	Commit  = "unknown"
)

// String returns a short version string suitable for `kapish version`.
// Order of preference:
//   1. info.Main.Version (when installed via `go install <module>@<tag>`)
//   2. VCS revision (+"+dirty" if dirty) from BuildInfo
//   3. ldflags-injected Version (Makefile builds)
//   4. "dev"
func String() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
		var rev, dirty string
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
				if len(rev) > 7 {
					rev = rev[:7]
				}
			case "vcs.modified":
				if s.Value == "true" {
					dirty = "+dirty"
				}
			}
		}
		if rev != "" {
			return rev + dirty
		}
	}
	if Version != "" && Version != "dev" {
		return Version
	}
	return "dev"
}

// Long is the human-readable version line for `kapish version`.
func Long() string {
	return fmt.Sprintf("kapish %s (commit %s)", String(), Commit)
}
```

- [ ] **Step 4: Run version package tests**

Run: `go test ./internal/version -v`
Expected: PASS for both tests.

- [ ] **Step 5: Wire `version` subcommand**

Create `cmd/kapish/version.go`:

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kapish version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(version.Long())
			return nil
		},
	}
}
```

- [ ] **Step 6: Register version command on root**

Modify `cmd/kapish/root.go` — add registration inside `newRootCmd`:

```go
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage: true,
	}
	root.AddCommand(newVersionCmd())
	return root
}
```

- [ ] **Step 7: Build and verify**

Run: `go build ./cmd/kapish && ./kapish version`
Expected: `kapish <something> (commit unknown)` — the `<something>` is either "dev", a short git SHA (possibly +dirty), or a semver.

- [ ] **Step 8: Commit**

```bash
git add internal/version/version.go internal/version/version_test.go cmd/kapish/version.go cmd/kapish/root.go
git commit -m "feat(cli): add version subcommand using BuildInfo"
```

---

## Task 4: Define Config types and built-in defaults

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/config/config.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultsMatchSpec(t *testing.T) {
	d := Defaults()

	// management clusters: empty entries, empty current
	assert.Equal(t, "", d.ManagementClusters.Current)
	assert.Empty(t, d.ManagementClusters.Entries)

	// shell defaults
	assert.Equal(t, "", d.Shell.Command, "Shell.Command default empty -> resolves to $SHELL at spawn")
	assert.Equal(t, "", d.Shell.Cwd)
	assert.Equal(t, "[{cluster}] ", d.Shell.Prompt)
	assert.NotNil(t, d.Shell.Env, "Env must be a non-nil map")
	assert.Empty(t, d.Shell.Env)
	assert.NotNil(t, d.Shell.Aliases)
	assert.Empty(t, d.Shell.Aliases)

	// ui defaults
	assert.Equal(t, "dark", d.UI.Theme)
	assert.Equal(t, 30, d.UI.RefreshIntervalSec)
	assert.False(t, d.UI.OneShot)

	// web defaults
	assert.Equal(t, 0, d.Web.DefaultPort)
	assert.True(t, d.Web.OpenBrowser)
	assert.Equal(t, "127.0.0.1", d.Web.BindAddr)
}

func TestDefaultsAreFresh(t *testing.T) {
	// Mutating one returned Defaults() must not affect the next.
	a := Defaults()
	a.Shell.Env["X"] = "1"
	a.Shell.Aliases["k"] = "kubectl"
	a.ManagementClusters.Entries = append(a.ManagementClusters.Entries, ManagementClusterEntry{Name: "x"})

	b := Defaults()
	require.Empty(t, b.Shell.Env, "second Defaults() must have empty Env")
	require.Empty(t, b.Shell.Aliases)
	require.Empty(t, b.ManagementClusters.Entries)
}
```

- [ ] **Step 2: Run the test, expect failure**

Run: `go get github.com/stretchr/testify@latest && go test ./internal/config -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement Config types**

Create `internal/config/config.go`:

```go
// Package config defines kapish's configuration types, defaults, and the
// load/validate/persist pipeline. The merge order is:
//   built-in Defaults() < config file < env vars < command-line flags
package config

// Config is the top-level kapish configuration.
type Config struct {
	ManagementClusters ManagementClustersConfig `yaml:"managementClusters"`
	Shell              ShellConfig              `yaml:"shell"`
	UI                 UIConfig                 `yaml:"ui"`
	Web                WebConfig                `yaml:"web"`
}

type ManagementClustersConfig struct {
	Current string                   `yaml:"current,omitempty"`
	Entries []ManagementClusterEntry `yaml:"entries,omitempty"`
}

type ManagementClusterEntry struct {
	Name       string `yaml:"name"`
	Kubeconfig string `yaml:"kubeconfig,omitempty"`
	Context    string `yaml:"context,omitempty"`
	Namespace  string `yaml:"namespace,omitempty"`
}

type ShellConfig struct {
	Command string            `yaml:"command,omitempty"`
	Cwd     string            `yaml:"cwd,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty"`
	Prompt  string            `yaml:"prompt,omitempty"`
}

type UIConfig struct {
	Theme              string `yaml:"theme"`
	RefreshIntervalSec int    `yaml:"refreshIntervalSec"`
	OneShot            bool   `yaml:"oneShot"`
}

type WebConfig struct {
	DefaultPort int    `yaml:"defaultPort"`
	OpenBrowser bool   `yaml:"openBrowser"`
	BindAddr    string `yaml:"bindAddr"`
}

// Defaults returns a fresh built-in Config. Each call returns an
// independent copy — callers can mutate the result freely.
func Defaults() Config {
	return Config{
		ManagementClusters: ManagementClustersConfig{
			Current: "",
			Entries: nil,
		},
		Shell: ShellConfig{
			Command: "",
			Cwd:     "",
			Env:     map[string]string{},
			Aliases: map[string]string{},
			Prompt:  "[{cluster}] ",
		},
		UI: UIConfig{
			Theme:              "dark",
			RefreshIntervalSec: 30,
			OneShot:            false,
		},
		Web: WebConfig{
			DefaultPort: 0,
			OpenBrowser: true,
			BindAddr:    "127.0.0.1",
		},
	}
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for `TestDefaultsMatchSpec` and `TestDefaultsAreFresh`.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go go.mod go.sum
git commit -m "feat(config): define Config types and defaults"
```

---

## Task 5: Resolve the config-file path

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/config/path.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/path_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/path_test.go`:

```go
package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePath_FlagWins(t *testing.T) {
	p, err := ResolvePath(PathSources{
		Flag:           "/tmp/explicit.yaml",
		EnvVar:         "/tmp/env.yaml",
		XDGConfigHome:  "/tmp/xdg",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/explicit.yaml", p)
}

func TestResolvePath_EnvWhenNoFlag(t *testing.T) {
	p, err := ResolvePath(PathSources{
		EnvVar:        "/tmp/env.yaml",
		XDGConfigHome: "/tmp/xdg",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/env.yaml", p)
}

func TestResolvePath_XDGWhenNoFlagNoEnv(t *testing.T) {
	p, err := ResolvePath(PathSources{
		XDGConfigHome: "/tmp/xdg",
	})
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp/xdg", "kapish", "config.yaml"), p)
}

func TestResolvePath_HomeFallback(t *testing.T) {
	p, err := ResolvePath(PathSources{
		Home: "/users/foo",
	})
	assert.NoError(t, err)
	assert.Equal(t, "/users/foo/.config/kapish/config.yaml", p)
}

func TestResolvePath_NoSources(t *testing.T) {
	_, err := ResolvePath(PathSources{})
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestResolvePath -v`
Expected: FAIL — `ResolvePath` and `PathSources` undefined.

- [ ] **Step 3: Implement path resolution**

Create `internal/config/path.go`:

```go
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
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS — all `TestResolvePath_*` tests pass plus existing config tests.

- [ ] **Step 5: Commit**

```bash
git add internal/config/path.go internal/config/path_test.go
git commit -m "feat(config): resolve config-file path with flag>env>XDG>home"
```

---

## Task 6: Load Config from a YAML file (read path)

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/config/load.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/load_test.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/testdata/valid_basic.yaml`
- Create: `/Users/varun/projects/personal/kapish/internal/config/testdata/invalid_syntax.yaml`

- [ ] **Step 1: Add yaml.v3 dependency**

Run: `go get gopkg.in/yaml.v3@latest`
Expected: `go: added gopkg.in/yaml.v3 v3.x.x`

- [ ] **Step 2: Write a valid testdata fixture**

Create `internal/config/testdata/valid_basic.yaml`:

```yaml
managementClusters:
  current: prod-mgmt
  entries:
    - name: prod-mgmt
      kubeconfig: /home/me/.kube/prod
      context: prod-admin

shell:
  command: /bin/zsh
  cwd: /tmp
  env:
    EDITOR: vim
  aliases:
    k: kubectl
  prompt: "[{cluster}] $ "

ui:
  theme: light
  refreshIntervalSec: 60
  oneShot: true

web:
  defaultPort: 8080
  openBrowser: false
  bindAddr: 127.0.0.1
```

- [ ] **Step 3: Write a deliberately broken testdata fixture**

Create `internal/config/testdata/invalid_syntax.yaml`:

```yaml
managementClusters:
  current: prod-mgmt
  entries
    - name: oops missing colon
```

- [ ] **Step 4: Write the failing test**

Create `internal/config/load_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile_ValidBasic(t *testing.T) {
	c, err := LoadFromFile("testdata/valid_basic.yaml")
	require.NoError(t, err)

	assert.Equal(t, "prod-mgmt", c.ManagementClusters.Current)
	require.Len(t, c.ManagementClusters.Entries, 1)
	assert.Equal(t, "prod-mgmt", c.ManagementClusters.Entries[0].Name)
	assert.Equal(t, "/home/me/.kube/prod", c.ManagementClusters.Entries[0].Kubeconfig)

	assert.Equal(t, "/bin/zsh", c.Shell.Command)
	assert.Equal(t, "/tmp", c.Shell.Cwd)
	assert.Equal(t, "vim", c.Shell.Env["EDITOR"])
	assert.Equal(t, "kubectl", c.Shell.Aliases["k"])
	assert.Equal(t, "[{cluster}] $ ", c.Shell.Prompt)

	assert.Equal(t, "light", c.UI.Theme)
	assert.Equal(t, 60, c.UI.RefreshIntervalSec)
	assert.True(t, c.UI.OneShot)

	assert.Equal(t, 8080, c.Web.DefaultPort)
	assert.False(t, c.Web.OpenBrowser)
}

func TestLoadFromFile_MissingFileReturnsDefaults(t *testing.T) {
	c, err := LoadFromFile("testdata/does-not-exist.yaml")
	require.NoError(t, err, "missing file is not an error; defaults are returned")
	d := Defaults()
	assert.Equal(t, d, c)
}

func TestLoadFromFile_SyntaxErrorReturnsError(t *testing.T) {
	_, err := LoadFromFile("testdata/invalid_syntax.yaml")
	require.Error(t, err)
}

// LoadFromFile must overlay file values on top of defaults: keys absent in
// the file keep their default values.
func TestLoadFromFile_OverlaysDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`ui:
  theme: light
`), 0o600))

	c, err := LoadFromFile(path)
	require.NoError(t, err)

	d := Defaults()
	assert.Equal(t, "light", c.UI.Theme, "file value applied")
	assert.Equal(t, d.UI.RefreshIntervalSec, c.UI.RefreshIntervalSec, "absent keys use defaults")
	assert.Equal(t, d.Web.BindAddr, c.Web.BindAddr)
	assert.Equal(t, d.Shell.Prompt, c.Shell.Prompt)
}
```

- [ ] **Step 5: Run tests, expect failure**

Run: `go test ./internal/config -run TestLoadFromFile -v`
Expected: FAIL — `LoadFromFile` undefined.

- [ ] **Step 6: Implement LoadFromFile**

Create `internal/config/load.go`:

```go
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
```

- [ ] **Step 7: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all four `TestLoadFromFile_*` tests.

- [ ] **Step 8: Commit**

```bash
git add internal/config/load.go internal/config/load_test.go internal/config/testdata/ go.mod go.sum
git commit -m "feat(config): load from YAML file overlaying defaults"
```

---

## Task 7: Apply env-var and flag overrides on top of file

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/internal/config/load.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/resolve_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/resolve_test.go`:

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyOverrides_FlagBeatsFileBeatsDefaults(t *testing.T) {
	c, err := LoadFromFile("testdata/valid_basic.yaml")
	require.NoError(t, err)

	// No flags: file values stand.
	got := ApplyOverrides(c, FlagOverrides{})
	assert.Equal(t, "prod-mgmt", got.ManagementClusters.Current)
	assert.Equal(t, "/home/me/.kube/prod", got.ManagementClusters.Entries[0].Kubeconfig)

	// --kubeconfig and --context flags override the current entry.
	got = ApplyOverrides(c, FlagOverrides{
		Kubeconfig: "/tmp/override.kubeconfig",
		Context:    "override-ctx",
	})
	assert.Equal(t, "/tmp/override.kubeconfig", got.ManagementClusters.Entries[0].Kubeconfig)
	assert.Equal(t, "override-ctx", got.ManagementClusters.Entries[0].Context)
}

func TestApplyOverrides_OneShotFlag(t *testing.T) {
	c := Defaults()
	got := ApplyOverrides(c, FlagOverrides{OneShot: boolPtr(true)})
	assert.True(t, got.UI.OneShot)

	c.UI.OneShot = true
	got = ApplyOverrides(c, FlagOverrides{OneShot: boolPtr(false)})
	assert.False(t, got.UI.OneShot, "explicit false flag must override file's true")
}

// When no managementClusters.current is set but flags supply kubeconfig/context,
// a synthetic single-entry list is created so downstream code always has one.
func TestApplyOverrides_SynthesizesEntryWhenAbsent(t *testing.T) {
	c := Defaults()
	got := ApplyOverrides(c, FlagOverrides{
		Kubeconfig: "/tmp/k",
		Context:    "ctx",
	})
	require.Len(t, got.ManagementClusters.Entries, 1)
	assert.Equal(t, "default", got.ManagementClusters.Entries[0].Name)
	assert.Equal(t, "default", got.ManagementClusters.Current)
	assert.Equal(t, "/tmp/k", got.ManagementClusters.Entries[0].Kubeconfig)
	assert.Equal(t, "ctx", got.ManagementClusters.Entries[0].Context)
}

func boolPtr(b bool) *bool { return &b }
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestApplyOverrides -v`
Expected: FAIL — `ApplyOverrides` and `FlagOverrides` undefined.

- [ ] **Step 3: Implement ApplyOverrides**

Append to `internal/config/load.go`:

```go
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
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS — all `TestApplyOverrides_*` plus existing tests.

- [ ] **Step 5: Commit**

```bash
git add internal/config/load.go internal/config/resolve_test.go
git commit -m "feat(config): apply flag overrides on top of file"
```

---

## Task 8: Validate shell config

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/config/validate.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/validate_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/validate_test.go`:

```go
package config

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateShell_EmptyCommandOK(t *testing.T) {
	// Empty command means "use $SHELL at spawn time" — accepted.
	errs := validateShell(ShellConfig{Command: ""})
	assert.Empty(t, errs)
}

func TestValidateShell_KnownShellInPath(t *testing.T) {
	// Pick a shell that is virtually always installed.
	bash, err := exec.LookPath("bash")
	require.NoError(t, err, "this test assumes bash is on PATH")

	errs := validateShell(ShellConfig{Command: bash})
	assert.Empty(t, errs)
}

func TestValidateShell_UnsupportedBasename(t *testing.T) {
	// /bin/ksh has a basename `ksh` which v1 doesn't support.
	errs := validateShell(ShellConfig{Command: "/usr/bin/ksh"})
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "unsupported shell")
}

func TestValidateShell_NotInPath(t *testing.T) {
	errs := validateShell(ShellConfig{Command: "/totally/not/a/real/zsh"})
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "not found")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestValidateShell -v`
Expected: FAIL — `validateShell` undefined.

- [ ] **Step 3: Implement validateShell**

Create `internal/config/validate.go`:

```go
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
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all `TestValidateShell_*` plus existing tests.

- [ ] **Step 5: Commit**

```bash
git add internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): validate shell config"
```

---

## Task 9: Validate env vars and aliases

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate.go`
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/config/validate_test.go`:

```go
func TestValidateEnv_KeyRules(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"upper letters", "KUBECONFIG", false},
		{"with underscore", "AWS_REGION", false},
		{"with digits", "FOO123", false},
		{"leading underscore", "_FOO", false},
		{"leading digit", "1FOO", true},
		{"lowercase", "foo", true},
		{"empty", "", true},
		{"spaces", "FOO BAR", true},
		{"hyphen", "FOO-BAR", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{tc.key: "v"}
			errs := validateEnv(env)
			if tc.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateAliases_NameRules(t *testing.T) {
	cases := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{"simple", "k", false},
		{"with underscore", "k_get", false},
		{"with digits", "k1", false},
		{"with hyphen", "k-get", false},
		{"leading digit", "1k", true},
		{"with space", "k get", true},
		{"with equals", "k=v", true},
		{"empty", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := map[string]string{tc.alias: "kubectl"}
			errs := validateAliases(a)
			if tc.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests, expect failure**

Run: `go test ./internal/config -run "TestValidateEnv_|TestValidateAliases_" -v`
Expected: FAIL — `validateEnv` and `validateAliases` undefined.

- [ ] **Step 3: Implement validateEnv and validateAliases**

Append to `internal/config/validate.go`:

```go
import (
	"regexp"
)

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
```

> **Note** — the `import` line shown above is incremental: merge it into the existing import block at the top of `validate.go` rather than adding a second `import (...)` clause.

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all `TestValidateEnv_*` and `TestValidateAliases_*` subtests plus existing.

- [ ] **Step 5: Commit**

```bash
git add internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): validate env vars and alias names"
```

---

## Task 10: Validate prompt template tokens

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate.go`
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/config/validate_test.go`:

```go
func TestValidatePrompt_KnownTokens(t *testing.T) {
	cases := []string{
		"",
		"$ ",
		"[{cluster}] ",
		"[{cluster}/{ns}] {ctx} {time} ",
		"({provider}) > ",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			errs := validatePrompt(p)
			assert.Empty(t, errs, "should accept: %q", p)
		})
	}
}

func TestValidatePrompt_UnknownToken(t *testing.T) {
	errs := validatePrompt("[{cluster}] {region} ")
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "{region}")
}

func TestValidatePrompt_MalformedToken(t *testing.T) {
	errs := validatePrompt("hello {cluster ")
	require.NotEmpty(t, errs)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestValidatePrompt -v`
Expected: FAIL — `validatePrompt` undefined.

- [ ] **Step 3: Implement validatePrompt**

Append to `internal/config/validate.go`:

```go
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
```

> **Note** — add `"strings"` to the imports in `validate.go`.

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all `TestValidatePrompt_*` plus existing.

- [ ] **Step 5: Commit**

```bash
git add internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): validate prompt template tokens"
```

---

## Task 11: Validate management cluster entries

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate.go`
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/config/validate_test.go`:

```go
func TestValidateMgmt_EmptyAccepted(t *testing.T) {
	errs := validateMgmt(ManagementClustersConfig{})
	assert.Empty(t, errs)
}

func TestValidateMgmt_DuplicateNames(t *testing.T) {
	m := ManagementClustersConfig{
		Entries: []ManagementClusterEntry{
			{Name: "a"},
			{Name: "a"},
		},
	}
	errs := validateMgmt(m)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "duplicate")
}

func TestValidateMgmt_EmptyEntryName(t *testing.T) {
	m := ManagementClustersConfig{
		Entries: []ManagementClusterEntry{{Name: ""}},
	}
	errs := validateMgmt(m)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "name")
}

func TestValidateMgmt_CurrentMustReferenceEntry(t *testing.T) {
	m := ManagementClustersConfig{
		Current: "missing",
		Entries: []ManagementClusterEntry{{Name: "a"}},
	}
	errs := validateMgmt(m)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "current")
	assert.Contains(t, errs[0].Error(), "missing")
}

func TestValidateMgmt_HappyPath(t *testing.T) {
	m := ManagementClustersConfig{
		Current: "a",
		Entries: []ManagementClusterEntry{
			{Name: "a"},
			{Name: "b"},
		},
	}
	errs := validateMgmt(m)
	assert.Empty(t, errs)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestValidateMgmt -v`
Expected: FAIL — `validateMgmt` undefined.

- [ ] **Step 3: Implement validateMgmt**

Append to `internal/config/validate.go`:

```go
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
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all `TestValidateMgmt_*` plus existing.

- [ ] **Step 5: Commit**

```bash
git add internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): validate managementClusters entries"
```

---

## Task 12: Aggregate validators into top-level Validate()

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate.go`
- Modify: `/Users/varun/projects/personal/kapish/internal/config/validate_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/config/validate_test.go`:

```go
func TestValidate_HappyDefaults(t *testing.T) {
	require.NoError(t, Validate(Defaults()))
}

func TestValidate_AggregatesErrors(t *testing.T) {
	c := Defaults()
	c.Shell.Env = map[string]string{"bad-key": "x"}        // env error
	c.Shell.Aliases = map[string]string{"1bad": "kubectl"} // alias error
	c.Shell.Prompt = "{nope}"                              // prompt error
	c.ManagementClusters.Current = "nada"                  // mgmt error

	err := Validate(c)
	require.Error(t, err)
	// Validate should surface all errors at once, joined.
	msg := err.Error()
	assert.Contains(t, msg, "env key")
	assert.Contains(t, msg, "alias name")
	assert.Contains(t, msg, "{nope}")
	assert.Contains(t, msg, "current")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestValidate -v`
Expected: FAIL — `Validate` undefined or aggregator missing.

- [ ] **Step 3: Implement Validate**

Append to `internal/config/validate.go`:

```go
// Validate runs every individual validator and returns a single error that
// joins all messages (errors.Join) so the user sees every problem at once,
// not just the first.
func Validate(c Config) error {
	var all []error
	all = append(all, validateShell(c.Shell)...)
	all = append(all, validateEnv(c.Shell.Env)...)
	all = append(all, validateAliases(c.Shell.Aliases)...)
	all = append(all, validatePrompt(c.Shell.Prompt)...)
	all = append(all, validateMgmt(c.ManagementClusters)...)
	if len(all) == 0 {
		return nil
	}
	return errors.Join(all...)
}
```

> **Note** — `errors.Join` requires Go 1.20+; we're on 1.22. Make sure `"errors"` is in the import block.

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS — all validation tests including aggregator.

- [ ] **Step 5: Commit**

```bash
git add internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): aggregate validators into Validate"
```

---

## Task 13: Comment-preserving YAML write via yaml.v3 Node API

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/config/write.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/write_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/write_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const inputWithComments = `# kapish config
# user comment that must survive a write

managementClusters:
  current: prod
  entries:
    - name: prod              # the prod mgmt cluster
      kubeconfig: /home/me/.kube/prod
      context: prod-admin

shell:
  command: /bin/zsh
  prompt: "[{cluster}] $ "    # template with cluster token
  env:
    EDITOR: vim               # I prefer vim
  aliases:
    k: kubectl

ui:
  theme: dark
  refreshIntervalSec: 30
  oneShot: false

web:
  defaultPort: 0
  openBrowser: true
  bindAddr: 127.0.0.1
`

func TestWriteToFile_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(inputWithComments), 0o600))

	// Load the config, change the theme, write it back.
	c, err := LoadFromFile(path)
	require.NoError(t, err)
	c.UI.Theme = "light"

	require.NoError(t, WriteToFile(path, c))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	got := string(out)

	// Comments preserved
	assert.Contains(t, got, "# kapish config")
	assert.Contains(t, got, "# user comment that must survive a write")
	assert.Contains(t, got, "# the prod mgmt cluster")
	assert.Contains(t, got, "# template with cluster token")
	assert.Contains(t, got, "# I prefer vim")

	// Theme value updated
	// We assert this via a second load round-trip to avoid coupling
	// the test to exact whitespace.
	c2, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c2.UI.Theme)
}

func TestWriteToFile_CreatesParentDirAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "kapish", "config.yaml")
	c := Defaults()
	c.UI.Theme = "light"

	require.NoError(t, WriteToFile(path, c))

	c2, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c2.UI.Theme)

	// Permissions: file 0600, parent dir 0700.
	fi, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	pi, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), pi.Mode().Perm())
}

func TestWriteToFile_NoExistingFile_StillWritesValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	c := Defaults()
	c.Shell.Env = map[string]string{"EDITOR": "vim"}

	require.NoError(t, WriteToFile(path, c))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	// produced YAML round-trips
	c2, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "vim", c2.Shell.Env["EDITOR"])
	assert.True(t, strings.Contains(string(got), "EDITOR"))
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestWriteToFile -v`
Expected: FAIL — `WriteToFile` undefined.

- [ ] **Step 3: Implement WriteToFile**

Create `internal/config/write.go`:

```go
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// WriteToFile writes c back to path. If the file already exists, the existing
// YAML's comments and key ordering are preserved by mutating a yaml.Node tree
// in place rather than re-marshaling from the Go struct. If the file does not
// exist, a fresh YAML document is generated from c.
//
// File mode is 0600; parent directories are created with 0700 if missing.
func WriteToFile(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}

	out, err := renderYAML(path, c)
	if err != nil {
		return err
	}

	// Atomic write via temp file + rename.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kapish-config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("config: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("config: write %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("config: chmod %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("config: close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("config: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

// renderYAML produces the bytes to write. If path exists, it loads the
// existing YAML as a yaml.Node tree, applies the values from c onto the
// tree (preserving comments and ordering for keys that already exist),
// and serializes. If path does not exist, it marshals c directly.
func renderYAML(path string, c Config) ([]byte, error) {
	existing, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// No existing file: marshal directly.
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(c); err != nil {
			return nil, fmt.Errorf("config: encode: %w", err)
		}
		_ = enc.Close()
		return buf.Bytes(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(existing, &root); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	// Re-marshal c into a yaml.Node so we can patch values back into root.
	var fresh yaml.Node
	if err := fresh.Encode(c); err != nil {
		return nil, fmt.Errorf("config: encode patch: %w", err)
	}

	// root is DocumentNode with .Content[0] = mapping; same for fresh.
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		patchMappingValues(root.Content[0], &fresh)
	} else {
		// Empty file or weird shape — replace wholesale.
		root = fresh
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, fmt.Errorf("config: encode: %w", err)
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}

// patchMappingValues walks the existing mapping node and replaces values
// found in fresh, preserving comments on the existing node. Keys present
// only in fresh are appended. Keys present only in existing are kept
// (so user-only keys aren't deleted).
func patchMappingValues(existing, fresh *yaml.Node) {
	if existing.Kind != yaml.MappingNode || fresh.Kind != yaml.MappingNode {
		// types diverged — replace existing's content with fresh's
		existing.Content = fresh.Content
		return
	}

	idx := indexMapping(existing)
	for i := 0; i < len(fresh.Content); i += 2 {
		key := fresh.Content[i].Value
		newVal := fresh.Content[i+1]
		if pos, ok := idx[key]; ok {
			existingVal := existing.Content[pos+1]
			if existingVal.Kind == yaml.MappingNode && newVal.Kind == yaml.MappingNode {
				patchMappingValues(existingVal, newVal)
			} else {
				// Replace the value but keep the existing key node (with its
				// head/foot comments).
				existing.Content[pos+1] = newVal
			}
		} else {
			existing.Content = append(existing.Content, fresh.Content[i], newVal)
		}
	}
}

func indexMapping(n *yaml.Node) map[string]int {
	out := make(map[string]int, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		out[n.Content[i].Value] = i
	}
	return out
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all `TestWriteToFile_*` plus existing.

- [ ] **Step 5: Commit**

```bash
git add internal/config/write.go internal/config/write_test.go
git commit -m "feat(config): comment-preserving YAML write"
```

---

## Task 14: File-locked writes

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/internal/config/write.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/lock_test.go`

- [ ] **Step 1: Add gofrs/flock**

Run: `go get github.com/gofrs/flock@latest`
Expected: `go: added github.com/gofrs/flock v0.x.x`

- [ ] **Step 2: Write the failing test**

Create `internal/config/lock_test.go`:

```go
package config

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrent writes must serialize: the final file must contain valid YAML
// with one of the writers' values, not interleaved bytes from both.
func TestWriteToFile_ConcurrentWritesSerialize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	c := Defaults()
	require.NoError(t, WriteToFile(path, c))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cc := Defaults()
			cc.UI.RefreshIntervalSec = 100 + i
			_ = WriteToFile(path, cc)
		}(i)
	}
	wg.Wait()

	// File must still parse — not corrupted by interleaving.
	got, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, got.UI.RefreshIntervalSec, 100)
	assert.Less(t, got.UI.RefreshIntervalSec, 120)
}
```

- [ ] **Step 3: Run test, observe baseline behavior**

Run: `go test ./internal/config -run TestWriteToFile_ConcurrentWritesSerialize -v`
Expected: PASS or FAIL — this is timing-sensitive. The point is to verify that, *with* the lock, the test reliably passes. Without the lock, it may pass intermittently. Proceed to Step 4 to add the lock.

- [ ] **Step 4: Wrap WriteToFile with a flock**

Modify `internal/config/write.go` — replace the body of `WriteToFile` with:

```go
import (
	"github.com/gofrs/flock"
)

func WriteToFile(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}

	lockPath := path + ".lock"
	lk := flock.New(lockPath)
	if err := lk.Lock(); err != nil {
		return fmt.Errorf("config: lock %s: %w", lockPath, err)
	}
	defer func() {
		_ = lk.Unlock()
		_ = os.Remove(lockPath)
	}()

	out, err := renderYAML(path, c)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".kapish-config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("config: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("config: write %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("config: chmod %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("config: close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("config: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}
```

> **Note** — merge the `"github.com/gofrs/flock"` import into the existing import block; do not add a duplicate `import (...)` clause.

- [ ] **Step 5: Run tests, expect pass**

Run: `go test ./internal/config -v -count=5`
Expected: PASS — running with `-count=5` flushes any flakiness; the locked write should pass every iteration.

- [ ] **Step 6: Commit**

```bash
git add internal/config/write.go internal/config/lock_test.go go.mod go.sum
git commit -m "feat(config): file-lock writes with gofrs/flock"
```

---

## Task 15: First-run config template

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/config/template.go`
- Create: `/Users/varun/projects/personal/kapish/internal/config/template_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/template_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureFirstRunTemplate_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kapish", "config.yaml")

	created, err := EnsureFirstRunTemplate(path)
	require.NoError(t, err)
	assert.True(t, created, "should report it created the file")

	got, err := os.ReadFile(path)
	require.NoError(t, err)

	// Has guiding comment header
	assert.Contains(t, string(got), "# kapish config")
	// Has every top-level section commented out so users see it
	assert.Contains(t, string(got), "managementClusters")
	assert.Contains(t, string(got), "shell")
	assert.Contains(t, string(got), "ui")
	assert.Contains(t, string(got), "web")

	// File parses as valid YAML and yields default values when loaded.
	c, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, Defaults(), c)
}

func TestEnsureFirstRunTemplate_NoOpWhenExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: light\n"), 0o600))

	created, err := EnsureFirstRunTemplate(path)
	require.NoError(t, err)
	assert.False(t, created)

	c, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c.UI.Theme, "existing file untouched")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/config -run TestEnsureFirstRunTemplate -v`
Expected: FAIL — `EnsureFirstRunTemplate` undefined.

- [ ] **Step 3: Implement EnsureFirstRunTemplate**

Create `internal/config/template.go`:

```go
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
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/config -v`
Expected: PASS for all `TestEnsureFirstRunTemplate_*`.

- [ ] **Step 5: Commit**

```bash
git add internal/config/template.go internal/config/template_test.go
git commit -m "feat(config): first-run config template"
```

---

## Task 16: Wire global flags on the cobra root

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/cmd/kapish/root.go`
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/flags.go`
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/flags_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/kapish/flags_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGlobalFlags_DefaultsAreSane(t *testing.T) {
	cmd := newRootCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	g, err := readGlobalFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, "", g.ConfigPath)
	assert.Equal(t, "", g.Kubeconfig)
	assert.Equal(t, "", g.Context)
	assert.Equal(t, "info", g.LogLevel)
	assert.Equal(t, "", g.LogFile)
	assert.False(t, g.OneShot)
}

func TestParseGlobalFlags_AllValuesSet(t *testing.T) {
	cmd := newRootCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--config", "/tmp/c.yaml",
		"--kubeconfig", "/tmp/k",
		"--context", "ctx",
		"--log-level", "debug",
		"--log-file", "/tmp/k.log",
		"--one-shot",
	}))

	g, err := readGlobalFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/c.yaml", g.ConfigPath)
	assert.Equal(t, "/tmp/k", g.Kubeconfig)
	assert.Equal(t, "ctx", g.Context)
	assert.Equal(t, "debug", g.LogLevel)
	assert.Equal(t, "/tmp/k.log", g.LogFile)
	assert.True(t, g.OneShot)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./cmd/kapish -v`
Expected: FAIL — `readGlobalFlags`/`GlobalFlags` undefined.

- [ ] **Step 3: Implement flags.go**

Create `cmd/kapish/flags.go`:

```go
package main

import "github.com/spf13/cobra"

// GlobalFlags is the typed read-out of the cobra root's persistent flags.
// All subcommands extract their settings via this struct so flag handling
// stays in one place.
type GlobalFlags struct {
	ConfigPath string
	Kubeconfig string
	Context    string
	LogLevel   string
	LogFile    string
	OneShot    bool
}

func registerGlobalFlags(cmd *cobra.Command) {
	pf := cmd.PersistentFlags()
	pf.String("config", "", "Path to kapish config (overrides $KAPISH_CONFIG and XDG defaults)")
	pf.String("kubeconfig", "", "Path to kubeconfig for the management cluster (overrides config)")
	pf.String("context", "", "kubeconfig context for the management cluster (overrides config)")
	pf.String("log-level", "info", "Log level: debug | info | warn | error")
	pf.String("log-file", "", "Log file path. Use '-' for stderr. Empty = $XDG_CACHE_HOME/kapish/kapish.log.")
	pf.Bool("one-shot", false, "TUI exits after first spawned shell exits, instead of returning to list")
}

func readGlobalFlags(cmd *cobra.Command) (GlobalFlags, error) {
	pf := cmd.Flags()
	g := GlobalFlags{}
	var err error
	if g.ConfigPath, err = pf.GetString("config"); err != nil {
		return g, err
	}
	if g.Kubeconfig, err = pf.GetString("kubeconfig"); err != nil {
		return g, err
	}
	if g.Context, err = pf.GetString("context"); err != nil {
		return g, err
	}
	if g.LogLevel, err = pf.GetString("log-level"); err != nil {
		return g, err
	}
	if g.LogFile, err = pf.GetString("log-file"); err != nil {
		return g, err
	}
	if g.OneShot, err = pf.GetBool("one-shot"); err != nil {
		return g, err
	}
	return g, nil
}
```

- [ ] **Step 4: Wire flag registration into newRootCmd**

Modify `cmd/kapish/root.go` so registration happens:

```go
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage: true,
	}
	registerGlobalFlags(root)
	root.AddCommand(newVersionCmd())
	return root
}
```

- [ ] **Step 5: Run tests, expect pass**

Run: `go test ./cmd/kapish -v`
Expected: PASS for both `TestParseGlobalFlags_*`.

- [ ] **Step 6: Commit**

```bash
git add cmd/kapish/flags.go cmd/kapish/flags_test.go cmd/kapish/root.go
git commit -m "feat(cli): wire global flags (config, kubeconfig, context, log, one-shot)"
```

---

## Task 17: `kapish config validate` subcommand

**Files:**
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/config.go`
- Create: `/Users/varun/projects/personal/kapish/cmd/kapish/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/kapish/config_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate_PrintsEffectiveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`ui:
  theme: light
`), 0o600))

	root := newRootCmd()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"config", "validate", "--config", path})
	require.NoError(t, root.Execute())

	got := stdout.String()
	// Effective config should contain the override.
	assert.Contains(t, got, "theme: light")
	// And the default for an unset value.
	assert.Contains(t, got, "refreshIntervalSec: 30")
}

func TestConfigValidate_FailsOnInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// alias name with space violates the alias regex.
	require.NoError(t, os.WriteFile(path, []byte(`shell:
  aliases:
    "bad alias": kubectl
`), 0o600))

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "validate", "--config", path})
	err := root.Execute()
	require.Error(t, err)
	assert.True(t, strings.Contains(buf.String(), "alias") || strings.Contains(err.Error(), "alias"))
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./cmd/kapish -v -run TestConfigValidate`
Expected: FAIL — `config validate` subcommand not wired.

- [ ] **Step 3: Implement config subcommand**

Create `cmd/kapish/config.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/v4run/kapish/internal/config"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Inspect or edit kapish configuration",
	}
	c.AddCommand(newConfigValidateCmd())
	return c
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate config and print the merged effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			path, err := config.ResolvePath(config.PathSources{
				Flag:          g.ConfigPath,
				EnvVar:        os.Getenv("KAPISH_CONFIG"),
				XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
				Home:          os.Getenv("HOME"),
			})
			if err != nil {
				return err
			}

			cfg, err := config.LoadFromFile(path)
			if err != nil {
				return err
			}

			cfg = config.ApplyOverrides(cfg, config.FlagOverrides{
				Kubeconfig: g.Kubeconfig,
				Context:    g.Context,
				OneShot:    boolPtrIfSet(cmd, "one-shot", g.OneShot),
			})

			if err := config.Validate(cfg); err != nil {
				return err
			}

			out, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "# Effective kapish config (path:", path+")")
			fmt.Fprint(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}

// boolPtrIfSet returns &val when the named flag was explicitly provided on
// the command line, nil otherwise.
func boolPtrIfSet(cmd *cobra.Command, name string, val bool) *bool {
	f := cmd.Flag(name)
	if f == nil || !f.Changed {
		return nil
	}
	return &val
}
```

- [ ] **Step 4: Register `config` on the root**

Modify `cmd/kapish/root.go`:

```go
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage: true,
	}
	registerGlobalFlags(root)
	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd())
	return root
}
```

- [ ] **Step 5: Run tests, expect pass**

Run: `go test ./... -v`
Expected: PASS — all tests across the repo, including the two new `TestConfigValidate_*`.

- [ ] **Step 6: Smoke-test by hand**

Run: `go build ./cmd/kapish && ./kapish config validate --config /tmp/no-such-config.yaml || true`
Expected: prints `# Effective kapish config (path: /tmp/no-such-config.yaml)` followed by the default config (because the file is missing → defaults are loaded).

- [ ] **Step 7: Commit**

```bash
git add cmd/kapish/config.go cmd/kapish/config_test.go cmd/kapish/root.go
git commit -m "feat(cli): kapish config validate subcommand"
```

---

## Task 18: `kapish config edit` subcommand

**Files:**
- Modify: `/Users/varun/projects/personal/kapish/cmd/kapish/config.go`
- Modify: `/Users/varun/projects/personal/kapish/cmd/kapish/config_test.go`

- [ ] **Step 1: Append failing tests**

Append to `cmd/kapish/config_test.go`:

```go
func TestConfigEdit_RunsEditorAndValidates(t *testing.T) {
	// Use a tiny shell script as the "editor" — when invoked it overwrites
	// the file with valid YAML. We stage a target file the editor will modify.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: dark\n"), 0o600))

	editor := filepath.Join(dir, "editor.sh")
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\n"+
		"cat > \"$1\" <<'EOF'\n"+
		"ui:\n  theme: light\nEOF\n"), 0o755))

	t.Setenv("EDITOR", editor)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "edit", "--config", path})
	require.NoError(t, root.Execute())

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "theme: light")
}

func TestConfigEdit_RejectsInvalidEditedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: dark\n"), 0o600))

	editor := filepath.Join(dir, "editor.sh")
	// Editor introduces an unknown prompt token to trigger validation failure.
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\n"+
		"cat > \"$1\" <<'EOF'\n"+
		"shell:\n  prompt: \"{not_a_token}\"\nEOF\n"), 0o755))

	t.Setenv("EDITOR", editor)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "edit", "--config", path})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "{not_a_token}")
}

func TestConfigEdit_CreatesFromTemplateWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kapish", "config.yaml")
	// Editor that no-ops, leaving the template intact.
	editor := filepath.Join(dir, "editor.sh")
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	t.Setenv("EDITOR", editor)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "edit", "--config", path})
	require.NoError(t, root.Execute())

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "# kapish config")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./cmd/kapish -v -run TestConfigEdit`
Expected: FAIL — `config edit` subcommand not wired.

- [ ] **Step 3: Implement config edit**

Append to `cmd/kapish/config.go`:

```go
import (
	"errors"
	"os/exec"
)

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the kapish config in $EDITOR; validate after save",
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			path, err := config.ResolvePath(config.PathSources{
				Flag:          g.ConfigPath,
				EnvVar:        os.Getenv("KAPISH_CONFIG"),
				XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
				Home:          os.Getenv("HOME"),
			})
			if err != nil {
				return err
			}

			// Make sure the file exists with a useful template if needed.
			if _, err := config.EnsureFirstRunTemplate(path); err != nil {
				return err
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				return errors.New("$EDITOR (or $VISUAL) is not set")
			}

			ed := exec.Command(editor, path)
			ed.Stdin = os.Stdin
			ed.Stdout = os.Stdout
			ed.Stderr = os.Stderr
			if err := ed.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			cfg, err := config.LoadFromFile(path)
			if err != nil {
				return err
			}
			if err := config.Validate(cfg); err != nil {
				return fmt.Errorf("edited config is invalid:\n%w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Saved:", path)
			return nil
		},
	}
}
```

> **Note** — merge the `"errors"` and `"os/exec"` imports with the existing `import (...)` clause in `config.go`.

- [ ] **Step 4: Register `edit` under `config`**

Modify `newConfigCmd` in `cmd/kapish/config.go`:

```go
func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Inspect or edit kapish configuration",
	}
	c.AddCommand(newConfigValidateCmd())
	c.AddCommand(newConfigEditCmd())
	return c
}
```

- [ ] **Step 5: Run tests, expect pass**

Run: `go test ./... -v`
Expected: PASS for all `TestConfigEdit_*` plus existing.

- [ ] **Step 6: Commit**

```bash
git add cmd/kapish/config.go cmd/kapish/config_test.go
git commit -m "feat(cli): kapish config edit subcommand"
```

---

## Task 19: Structured logging via slog

**Files:**
- Create: `/Users/varun/projects/personal/kapish/internal/kapishlog/log.go`
- Create: `/Users/varun/projects/personal/kapish/internal/kapishlog/log_test.go`
- Modify: `/Users/varun/projects/personal/kapish/cmd/kapish/root.go`

- [ ] **Step 1: Add lumberjack**

Run: `go get gopkg.in/natefinch/lumberjack.v2@latest`
Expected: `go: added gopkg.in/natefinch/lumberjack.v2 v2.x.x`

- [ ] **Step 2: Write the failing test**

Create `internal/kapishlog/log_test.go`:

```go
package kapishlog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_LevelFiltersBelow(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Options{Level: "info", Writer: &buf})
	require.NoError(t, err)
	l.Debug("nope")
	l.Info("yes")

	out := buf.String()
	assert.NotContains(t, out, "nope")
	assert.Contains(t, out, "yes")
}

func TestNew_DebugLevelLetsDebugThrough(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Options{Level: "debug", Writer: &buf})
	require.NoError(t, err)
	l.Debug("hello-debug")
	assert.Contains(t, buf.String(), "hello-debug")
}

func TestNew_BadLevel(t *testing.T) {
	_, err := New(Options{Level: "loud"})
	require.Error(t, err)
}

func TestNew_ProducesJSON(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Options{Level: "info", Writer: &buf})
	require.NoError(t, err)
	l.Info("event", "k", "v")

	// Each output line must be a valid JSON object.
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m), "line: %s", line)
		assert.Equal(t, "INFO", m["level"])
		assert.Equal(t, "event", m["msg"])
		assert.Equal(t, "v", m["k"])
	}
}
```

- [ ] **Step 3: Run test, expect failure**

Run: `go test ./internal/kapishlog -v`
Expected: FAIL — package undefined.

- [ ] **Step 4: Implement kapishlog**

Create `internal/kapishlog/log.go`:

```go
// Package kapishlog wires log/slog with sensible defaults: JSON output,
// configurable level, optional rotated file output via lumberjack.
package kapishlog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Options control logger creation.
type Options struct {
	// Level is one of "debug", "info", "warn", "error". Empty is treated as "info".
	Level string

	// Writer is where logs go. If nil, FilePath is used; if both are empty,
	// logs go to stderr.
	Writer io.Writer

	// FilePath, when set and Writer is nil, opens a rotating log file at
	// that path. "-" means stderr (useful for `--log-file -`).
	FilePath string
}

// New returns a *slog.Logger configured per opts.
func New(opts Options) (*slog.Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	w := opts.Writer
	if w == nil {
		switch opts.FilePath {
		case "", "-":
			w = os.Stderr
		default:
			if err := os.MkdirAll(filepath.Dir(opts.FilePath), 0o700); err != nil {
				return nil, fmt.Errorf("kapishlog: mkdir %s: %w", filepath.Dir(opts.FilePath), err)
			}
			w = &lumberjack.Logger{
				Filename:   opts.FilePath,
				MaxSize:    10, // MB
				MaxBackups: 3,
				LocalTime:  true,
				Compress:   false,
			}
		}
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(h), nil
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, errors.New("kapishlog: unknown log level " + s)
	}
}
```

- [ ] **Step 5: Run tests, expect pass**

Run: `go test ./internal/kapishlog -v`
Expected: PASS for all `TestNew_*`.

- [ ] **Step 6: Wire logger into root command's PersistentPreRunE**

Modify `cmd/kapish/root.go`:

```go
package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/kapishlog"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			g, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			path := g.LogFile
			if path == "" {
				cache := os.Getenv("XDG_CACHE_HOME")
				if cache == "" {
					cache = filepath.Join(os.Getenv("HOME"), ".cache")
				}
				path = filepath.Join(cache, "kapish", "kapish.log")
			}
			logger, err := kapishlog.New(kapishlog.Options{Level: g.LogLevel, FilePath: path})
			if err != nil {
				return err
			}
			slog.SetDefault(logger)
			return nil
		},
	}
	registerGlobalFlags(root)
	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd())
	return root
}
```

- [ ] **Step 7: Build and smoke-test**

Run: `go build ./cmd/kapish && ./kapish version --log-level debug --log-file -`
Expected: prints version line to stdout; the JSON log line goes to stderr (don't see it duplicated, but no error either).

- [ ] **Step 8: Commit**

```bash
git add internal/kapishlog/log.go internal/kapishlog/log_test.go cmd/kapish/root.go go.mod go.sum
git commit -m "feat(log): structured slog logger with optional rotated file"
```

---

## Task 20: Makefile, README polish, and final verification

**Files:**
- Create: `/Users/varun/projects/personal/kapish/Makefile`
- Modify: `/Users/varun/projects/personal/kapish/README.md`

- [ ] **Step 1: Write the Makefile**

Create `Makefile`:

```makefile
GO       ?= go
PKG      := github.com/v4run/kapish
BINDIR   := bin
BIN      := $(BINDIR)/kapish

VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS  := -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT)

.PHONY: all build install test lint fmt tidy clean

all: build

build:
	@mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/kapish

install:
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/kapish

test:
	$(GO) test ./... -count=1

lint:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BINDIR)
```

- [ ] **Step 2: Smoke-test Makefile targets**

Run: `make tidy && make fmt && make lint && make test && make build && ./bin/kapish version`
Expected: all targets succeed; the version line prints and includes a non-`dev` value when a git tag/commit exists, or `dev` if not.

- [ ] **Step 3: Polish the README**

Replace `README.md` with:

````markdown
# kapish

A debugging tool for [Cluster API (CAPI)](https://cluster-api.sigs.k8s.io/). Lists workload clusters from a management cluster and drops you into a shell whose `KUBECONFIG` is pre-set, with configurable env vars, aliases, working directory, and prompt prefix. Runs as a TUI, or as a localhost web app with an in-browser terminal.

## Install

```sh
go install github.com/v4run/kapish/cmd/kapish@latest
```

Make sure `$(go env GOBIN)` (or `$(go env GOPATH)/bin` if `GOBIN` is unset) is in `PATH`.

Or build from source:

```sh
git clone https://github.com/v4run/kapish
cd kapish
make build
./bin/kapish version
```

## Usage

```sh
kapish                    # TUI (lands when implemented in Plan 3)
kapish serve              # web UI (Plan 4 + Plan 5)
kapish version            # show version
kapish config validate    # show merged effective config
kapish config edit        # edit config in $EDITOR; validate on save
```

### Global flags

| Flag | Purpose |
|---|---|
| `--config <path>` | override config-file path (also `$KAPISH_CONFIG`) |
| `--kubeconfig <path>` | override kubeconfig for the current management cluster |
| `--context <name>` | override kubeconfig context for the current management cluster |
| `--log-level debug\|info\|warn\|error` | log verbosity (default `info`) |
| `--log-file <path>` | log file (default `$XDG_CACHE_HOME/kapish/kapish.log`; `-` = stderr) |
| `--one-shot` | TUI exits after first spawned shell exits |

## Configuration

Config lives at `~/.config/kapish/config.yaml` by default. On first run, kapish writes a commented template there; edit it with:

```sh
kapish config edit
```

See [`docs/superpowers/specs/2026-05-09-kapish-design.md`](docs/superpowers/specs/2026-05-09-kapish-design.md) for the full schema.

## Status

In active development. See `docs/superpowers/plans/` for what's currently being built.
````

- [ ] **Step 4: Run the full test suite one more time**

Run: `make test`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add Makefile README.md
git commit -m "chore: add Makefile and polish README"
```

- [ ] **Step 6: Final smoke test of every entry point**

Run, one at a time:

```sh
./bin/kapish --help
./bin/kapish version
./bin/kapish config --help
./bin/kapish config validate
./bin/kapish config validate --config /tmp/no-such-config.yaml
EDITOR=true ./bin/kapish config edit --config /tmp/kapish-smoke.yaml
go install ./cmd/kapish && "$(go env GOPATH)/bin/kapish" version
```

Expected:
- `--help` prints usage including the `version` and `config` subcommands.
- `version` prints `kapish <ver> (commit <sha>)` (or `kapish dev (commit unknown)` if no git info).
- `config --help` lists `validate` and `edit`.
- `config validate` prints the effective config including defaults.
- `config validate --config /tmp/no-such-config.yaml` prints defaults (file missing → defaults).
- `config edit ...` succeeds (because `true` exits 0 immediately without modifying the file) — and the file ends up as the first-run template.
- `go install` puts `kapish` on `$GOBIN`/`$GOPATH/bin` and the installed binary works.

If anything misbehaves, fix and re-commit before moving to Plan 2.

---

## Plan-1 exit criteria

- [ ] `go install github.com/v4run/kapish/cmd/kapish@latest` works (locally before push, via the Makefile path; once pushed, the literal command works).
- [ ] `kapish version`, `kapish config validate`, `kapish config edit` all functional.
- [ ] Global flags wired and reachable from any subcommand.
- [ ] `make test` is green.
- [ ] First-run template lands at `$XDG_CONFIG_HOME/kapish/config.yaml` on first edit.
- [ ] `go vet ./...` is clean.

When all boxes are checked, Plan 1 is done. Plan 2 (Core libraries: capi + shell) is the next phase.
