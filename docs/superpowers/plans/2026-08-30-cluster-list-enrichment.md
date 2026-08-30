# Cluster List Enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the cluster list show extra columns whose values are joined in from a configurable shell/CLI command, fetched asynchronously and cached, without ever blocking or crashing the list.

**Architecture:** A new decoupled package `internal/enrich` owns a `Provider` interface, a `Manager` that fetches provider rows on per-provider tickers into a mutex-guarded cache, and a render-time join (`cluster.<joinOn>` → provider row → field) producing `Cell{Text, State}`. The shell provider execs a configured argv (no shell interpolation) and parses a JSON array. The TUI subscribes to `Manager.Updates()` and renders enrichment columns after the built-in ones. `capi.Cluster` is not modified.

**Tech Stack:** Go 1.26, bubbletea/lipgloss (TUI), `os/exec`, `encoding/json`, `log/slog`, testify.

## Global Constraints

- Go module: `github.com/v4run/kapish`; Go 1.26.0.
- Enrichment is **additive**: a malformed enrichment block must never prevent the list from working — `config validate` fails on it, but runtime code logs (`slog.Warn`) and skips.
- The shell provider runs argv **directly** via `exec.CommandContext` — never `sh -c`. No shell interpolation.
- `capi.Cluster` must not gain enrichment fields; the join happens in the render layer.
- `internal/enrich` imports neither `internal/capi` nor `internal/tui`/`internal/web`.
- Cell state text markers: Ready → value; Pending → `…`; Error → `!`; Empty → `-`. Pending/Error/Empty are styled dim/red per existing `style.go` styles.
- MVP scope: shell provider + TUI rendering only. No web columns, no HTTP/labels/in-cluster providers, JSON output only, `joinOn ∈ {name, namespace}`.
- Test framework: `github.com/stretchr/testify` (`assert`/`require`), matching existing tests.

---

### Task 1: enrich core types + Manager join/state logic

**Files:**
- Create: `internal/enrich/types.go`
- Create: `internal/enrich/provider.go`
- Create: `internal/enrich/manager.go`
- Test: `internal/enrich/manager_test.go`

**Interfaces:**
- Consumes: nothing (leaf package).
- Produces:
  - `type ClusterRef struct { Name, Namespace string }`
  - `type Row map[string]string`
  - `type Provider interface { Name() string; Fetch(ctx context.Context) (map[string]Row, error) }`
  - `type ProviderSpec struct { Provider Provider; Interval, Timeout time.Duration }`
  - `type Column struct { Header, Provider, Field, JoinOn string; Width int }`
  - `type CellState int` with consts `CellPending`, `CellReady`, `CellError`, `CellEmpty` (`CellPending` is the zero value).
  - `type Cell struct { Text string; State CellState }`
  - `type StaticProvider struct{...}`; `func NewStaticProvider(name string, rows map[string]Row) *StaticProvider`
  - `func NewManager(specs []ProviderSpec, columns []Column) *Manager`
  - `func (m *Manager) Columns() []Column`
  - `func (m *Manager) Cells(ref ClusterRef) []Cell` — length always `== len(Columns())`
  - `func (m *Manager) ProviderErrors() map[string]error`
  - `func (m *Manager) Updates() <-chan struct{}`
  - `func (m *Manager) refreshOnce(ctx context.Context, name string)` (unexported; same-package tests use it)

- [ ] **Step 1: Write the failing test**

Create `internal/enrich/manager_test.go`:

```go
package enrich

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// erroringProvider always fails its Fetch.
type erroringProvider struct{ name string }

func (p erroringProvider) Name() string { return p.name }
func (p erroringProvider) Fetch(context.Context) (map[string]Row, error) {
	return nil, context.DeadlineExceeded
}

func testManager() *Manager {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("cost", map[string]Row{
			"prod-eu-1": {"monthlyCost": "$412", "nodes": "12"},
		}),
		Interval: time.Minute,
		Timeout:  time.Second,
	}}
	cols := []Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	return NewManager(specs, cols)
}

func TestCells_PendingBeforeFetch(t *testing.T) {
	m := testManager()
	cells := m.Cells(ClusterRef{Name: "prod-eu-1"})
	require.Len(t, cells, 1)
	assert.Equal(t, CellPending, cells[0].State)
}

func TestCells_ReadyAfterFetch(t *testing.T) {
	m := testManager()
	m.refreshOnce(context.Background(), "cost")
	cells := m.Cells(ClusterRef{Name: "prod-eu-1"})
	assert.Equal(t, CellReady, cells[0].State)
	assert.Equal(t, "$412", cells[0].Text)
}

func TestCells_EmptyWhenNoMatch(t *testing.T) {
	m := testManager()
	m.refreshOnce(context.Background(), "cost")
	cells := m.Cells(ClusterRef{Name: "does-not-exist"})
	assert.Equal(t, CellEmpty, cells[0].State)
}

func TestCells_ErrorWhenFirstFetchFails(t *testing.T) {
	specs := []ProviderSpec{{Provider: erroringProvider{name: "cost"}, Interval: time.Minute, Timeout: time.Second}}
	cols := []Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	m := NewManager(specs, cols)
	m.refreshOnce(context.Background(), "cost")
	cells := m.Cells(ClusterRef{Name: "prod-eu-1"})
	assert.Equal(t, CellError, cells[0].State)
	assert.NotNil(t, m.ProviderErrors()["cost"])
}

func TestCells_JoinOnNamespace(t *testing.T) {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("team", map[string]Row{"prod": {"owner": "platform"}}),
		Interval: time.Minute, Timeout: time.Second,
	}}
	cols := []Column{{Header: "OWNER", Provider: "team", Field: "owner", JoinOn: "namespace"}}
	m := NewManager(specs, cols)
	m.refreshOnce(context.Background(), "team")
	cells := m.Cells(ClusterRef{Name: "prod-eu-1", Namespace: "prod"})
	assert.Equal(t, CellReady, cells[0].State)
	assert.Equal(t, "platform", cells[0].Text)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/enrich/ -run TestCells`
Expected: FAIL — build error, `NewManager`/`NewStaticProvider` undefined.

- [ ] **Step 3: Write the types**

Create `internal/enrich/types.go`:

```go
// Package enrich joins extra columns onto the cluster list from configurable
// providers. It depends on neither the capi package nor any UI package: the
// Manager is fed a lightweight ClusterRef and returns render-ready Cells.
package enrich

import "context"

// ClusterRef is the minimal cluster identity the manager needs to join on.
type ClusterRef struct {
	Name      string
	Namespace string
}

// Row is one provider output record: field name -> stringified value.
type Row map[string]string

// Provider fetches all rows for one source, once per refresh, keyed by the
// join value.
type Provider interface {
	Name() string
	Fetch(ctx context.Context) (map[string]Row, error)
}

// ProviderSpec pairs a provider with its refresh cadence and per-fetch timeout.
type ProviderSpec struct {
	Provider Provider
	Interval time.Duration
	Timeout  time.Duration
}

// Column is one configured enrichment column.
type Column struct {
	Header   string // display header
	Provider string // provider name this column reads from
	Field    string // which field of the joined Row to display
	JoinOn   string // cluster field matched against the provider row key: name|namespace
	Width    int    // optional min width; 0 = auto-size
}

// CellState classifies an enrichment cell. CellPending is the zero value so an
// unfetched provider naturally reads as pending.
type CellState int

const (
	CellPending CellState = iota
	CellReady
	CellError
	CellEmpty
)

// Cell is a render-ready enrichment value for one cluster/column.
type Cell struct {
	Text  string
	State CellState
}
```

Add the missing `time` import — replace `import "context"` with:

```go
import (
	"context"
	"time"
)
```

- [ ] **Step 4: Write StaticProvider**

Create `internal/enrich/provider.go`:

```go
package enrich

import "context"

// StaticProvider returns a fixed set of rows. Useful for tests and as the shape
// the (future) labels provider will follow.
type StaticProvider struct {
	name string
	rows map[string]Row
}

func NewStaticProvider(name string, rows map[string]Row) *StaticProvider {
	return &StaticProvider{name: name, rows: rows}
}

func (p *StaticProvider) Name() string { return p.name }

func (p *StaticProvider) Fetch(context.Context) (map[string]Row, error) {
	return p.rows, nil
}
```

- [ ] **Step 5: Write the Manager**

Create `internal/enrich/manager.go`:

```go
package enrich

import (
	"context"
	"sync"
)

type providerState struct {
	rows        map[string]Row
	everFetched bool
	lastErr     error
}

// Manager owns the enrichment providers and a cache of their last results, and
// computes render-ready Cells by joining that cache onto clusters.
type Manager struct {
	specs   map[string]ProviderSpec
	columns []Column

	mu    sync.RWMutex
	state map[string]*providerState

	updates chan struct{}
}

func NewManager(specs []ProviderSpec, columns []Column) *Manager {
	m := &Manager{
		specs:   make(map[string]ProviderSpec, len(specs)),
		columns: columns,
		state:   make(map[string]*providerState, len(specs)),
		updates: make(chan struct{}, 1),
	}
	for _, s := range specs {
		name := s.Provider.Name()
		m.specs[name] = s
		m.state[name] = &providerState{}
	}
	return m
}

// Columns returns the configured columns in display order.
func (m *Manager) Columns() []Column { return m.columns }

// Updates fires (coalesced, non-blocking) whenever a provider's cache changes.
func (m *Manager) Updates() <-chan struct{} { return m.updates }

func (m *Manager) signal() {
	select {
	case m.updates <- struct{}{}:
	default:
	}
}

// Cells returns one Cell per configured column, in column order.
func (m *Manager) Cells(ref ClusterRef) []Cell {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cells := make([]Cell, len(m.columns))
	for i, col := range m.columns {
		st := m.state[col.Provider]
		if st == nil {
			cells[i] = Cell{State: CellError}
			continue
		}
		if !st.everFetched {
			if st.lastErr != nil {
				cells[i] = Cell{State: CellError}
			} else {
				cells[i] = Cell{State: CellPending}
			}
			continue
		}
		key := ref.Name
		if col.JoinOn == "namespace" {
			key = ref.Namespace
		}
		row, ok := st.rows[key]
		if !ok {
			cells[i] = Cell{State: CellEmpty}
			continue
		}
		cells[i] = Cell{Text: row[col.Field], State: CellReady}
	}
	return cells
}

// ProviderErrors returns the last fetch error per provider name (absent if the
// last fetch succeeded).
func (m *Manager) ProviderErrors() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]error)
	for name, st := range m.state {
		if st.lastErr != nil {
			out[name] = st.lastErr
		}
	}
	return out
}

// refreshOnce fetches a single provider and updates its cache. On error the
// last good rows are kept so stale-but-present data keeps rendering.
func (m *Manager) refreshOnce(ctx context.Context, name string) {
	spec, ok := m.specs[name]
	if !ok {
		return
	}
	fctx := ctx
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		fctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}
	rows, err := spec.Provider.Fetch(fctx)

	m.mu.Lock()
	st := m.state[name]
	if err != nil {
		st.lastErr = err
	} else {
		st.rows = rows
		st.everFetched = true
		st.lastErr = nil
	}
	m.mu.Unlock()

	m.signal()
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/enrich/ -run TestCells`
Expected: PASS (5 tests).

- [ ] **Step 7: Commit**

```bash
git add internal/enrich/types.go internal/enrich/provider.go internal/enrich/manager.go internal/enrich/manager_test.go
git commit -m "feat(enrich): core types and manager join/state logic"
```

---

### Task 2: Manager async refresh loop

**Files:**
- Modify: `internal/enrich/manager.go`
- Test: `internal/enrich/manager_async_test.go`

**Interfaces:**
- Consumes: `Manager`, `ProviderSpec`, `refreshOnce` from Task 1.
- Produces:
  - `func (m *Manager) Start(ctx context.Context)` — launches one goroutine per provider; each fetches immediately, then on its `Interval`; all stop when `ctx` is done.
  - `func (m *Manager) RefreshAll(ctx context.Context)` — synchronously refreshes every provider once (used for warm-up and tests).

- [ ] **Step 1: Write the failing test**

Create `internal/enrich/manager_async_test.go`:

```go
package enrich

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart_FillsCellsAndSignals(t *testing.T) {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("cost", map[string]Row{"c1": {"v": "42"}}),
		Interval: time.Hour, // only the immediate first fetch matters here
		Timeout:  time.Second,
	}}
	cols := []Column{{Header: "V", Provider: "cost", Field: "v", JoinOn: "name"}}
	m := NewManager(specs, cols)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	select {
	case <-m.Updates():
	case <-time.After(2 * time.Second):
		t.Fatal("expected an update signal after Start")
	}

	cells := m.Cells(ClusterRef{Name: "c1"})
	require.Len(t, cells, 1)
	assert.Equal(t, CellReady, cells[0].State)
	assert.Equal(t, "42", cells[0].Text)
}

func TestRefreshAll(t *testing.T) {
	specs := []ProviderSpec{{
		Provider: NewStaticProvider("cost", map[string]Row{"c1": {"v": "9"}}),
		Interval: time.Hour, Timeout: time.Second,
	}}
	cols := []Column{{Header: "V", Provider: "cost", Field: "v", JoinOn: "name"}}
	m := NewManager(specs, cols)
	m.RefreshAll(context.Background())
	assert.Equal(t, CellReady, m.Cells(ClusterRef{Name: "c1"})[0].State)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/enrich/ -run 'TestStart|TestRefreshAll'`
Expected: FAIL — `Start`/`RefreshAll` undefined.

- [ ] **Step 3: Implement Start and RefreshAll**

Append to `internal/enrich/manager.go`:

```go
// Start launches a refresh goroutine per provider. Each fetches immediately,
// then on its Interval, until ctx is canceled.
func (m *Manager) Start(ctx context.Context) {
	for name, spec := range m.specs {
		go m.runProvider(ctx, name, spec.Interval)
	}
}

// RefreshAll refreshes every provider once, synchronously.
func (m *Manager) RefreshAll(ctx context.Context) {
	for name := range m.specs {
		m.refreshOnce(ctx, name)
	}
}

func (m *Manager) runProvider(ctx context.Context, name string, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	m.refreshOnce(ctx, name)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.refreshOnce(ctx, name)
		}
	}
}
```

Add `"time"` to the `manager.go` import block:

```go
import (
	"context"
	"sync"
	"time"
)
```

- [ ] **Step 4: Run tests (with the race detector) to verify they pass**

Run: `go test -race ./internal/enrich/`
Expected: PASS (all enrich tests, no data races).

- [ ] **Step 5: Commit**

```bash
git add internal/enrich/manager.go internal/enrich/manager_async_test.go
git commit -m "feat(enrich): async per-provider refresh loop"
```

---

### Task 3: Shell provider

**Files:**
- Create: `internal/enrich/shell.go`
- Test: `internal/enrich/shell_test.go`

**Interfaces:**
- Consumes: `Provider`, `Row` from Task 1.
- Produces:
  - `func NewShellProvider(name string, argv []string, keyField string) *ShellProvider`
  - `ShellProvider` implements `Provider`.

- [ ] **Step 1: Write the failing test**

Create `internal/enrich/shell_test.go`:

```go
package enrich

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
	return p
}

func TestShell_ParsesJSONArray(t *testing.T) {
	path := writeTemp(t, `[{"id":"c1","cost":"$412","nodes":12,"ok":true},{"id":"c2","cost":"$99"}]`)
	p := NewShellProvider("cost", []string{"cat", path}, "id")
	rows, err := p.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "$412", rows["c1"]["cost"])
	assert.Equal(t, "12", rows["c1"]["nodes"], "JSON numbers stringify without .0")
	assert.Equal(t, "true", rows["c1"]["ok"])
	assert.Equal(t, "$99", rows["c2"]["cost"])
}

func TestShell_SkipsRowsMissingKey(t *testing.T) {
	path := writeTemp(t, `[{"id":"c1","v":"a"},{"v":"b"},{"id":"c1","v":"dup"}]`)
	p := NewShellProvider("x", []string{"cat", path}, "id")
	rows, err := p.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1, "missing-key row skipped, duplicate-key row skipped")
	assert.Equal(t, "a", rows["c1"]["v"], "first wins on duplicate key")
}

func TestShell_MalformedJSONIsError(t *testing.T) {
	p := NewShellProvider("x", []string{"printf", "not json"}, "id")
	_, err := p.Fetch(context.Background())
	assert.Error(t, err)
}

func TestShell_NonZeroExitIsError(t *testing.T) {
	p := NewShellProvider("x", []string{"false"}, "id")
	_, err := p.Fetch(context.Background())
	assert.Error(t, err)
}

func TestShell_TimeoutIsError(t *testing.T) {
	p := NewShellProvider("x", []string{"sleep", "5"}, "id")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := p.Fetch(ctx)
	assert.Error(t, err)
}

func TestShell_EmptyCommandIsError(t *testing.T) {
	p := NewShellProvider("x", nil, "id")
	_, err := p.Fetch(context.Background())
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/enrich/ -run TestShell`
Expected: FAIL — `NewShellProvider` undefined.

- [ ] **Step 3: Implement the shell provider**

Create `internal/enrich/shell.go`:

```go
package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// ShellProvider execs a configured argv and parses its stdout as a JSON array
// of flat objects. It never runs through a shell — argv is executed directly,
// so there is no shell-injection surface.
type ShellProvider struct {
	name     string
	argv     []string
	keyField string
}

func NewShellProvider(name string, argv []string, keyField string) *ShellProvider {
	return &ShellProvider{name: name, argv: argv, keyField: keyField}
}

func (p *ShellProvider) Name() string { return p.name }

func (p *ShellProvider) Fetch(ctx context.Context) (map[string]Row, error) {
	if len(p.argv) == 0 {
		return nil, fmt.Errorf("enrich: shell provider %q: empty command", p.name)
	}
	cmd := exec.CommandContext(ctx, p.argv[0], p.argv[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("enrich: shell provider %q: %w", p.name, err)
	}

	var records []map[string]any
	if err := json.Unmarshal(out, &records); err != nil {
		return nil, fmt.Errorf("enrich: shell provider %q: parse json: %w", p.name, err)
	}

	rows := make(map[string]Row, len(records))
	for _, rec := range records {
		kv, ok := rec[p.keyField]
		if !ok {
			continue // missing key: skip
		}
		key := stringify(kv)
		if key == "" {
			continue
		}
		if _, dup := rows[key]; dup {
			continue // duplicate key: first wins
		}
		row := make(Row, len(rec))
		for k, v := range rec {
			row[k] = stringify(v)
		}
		rows[key] = row
	}
	return rows, nil
}

// stringify renders a decoded JSON value as display text. JSON numbers decode
// as float64; integral values render without a trailing ".0".
func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/enrich/ -run TestShell`
Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/enrich/shell.go internal/enrich/shell_test.go
git commit -m "feat(enrich): shell provider (argv exec + JSON parse)"
```

---

### Task 4: Config schema + validation

**Files:**
- Modify: `internal/config/config.go` (add structs + `Config.Enrichment` field)
- Modify: `internal/config/validate.go` (add `validateEnrichment`, call it in `Validate`)
- Test: `internal/config/validate_test.go` (append cases)

**Interfaces:**
- Consumes: existing `Config`, `Validate` from the config package.
- Produces:
  - `Config.Enrichment EnrichmentConfig`
  - `type EnrichmentConfig struct { Providers []EnrichProvider; Columns []EnrichColumn }`
  - `type EnrichProvider struct { Name, Type string; Command []string; Format, Key string; RefreshIntervalSec, TimeoutSec int }`
  - `type EnrichColumn struct { Header, Provider, Field, JoinOn string; Width int }`

- [ ] **Step 1: Write the failing test**

Append to `internal/config/validate_test.go`:

```go
func validEnrichment() EnrichmentConfig {
	return EnrichmentConfig{
		Providers: []EnrichProvider{{
			Name: "cost", Type: "shell", Command: []string{"kapish-cost", "--all"},
			Format: "json", Key: "clusterId", RefreshIntervalSec: 300, TimeoutSec: 15,
		}},
		Columns: []EnrichColumn{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}},
	}
}

func TestValidate_EnrichmentOK(t *testing.T) {
	c := Defaults()
	c.Enrichment = validEnrichment()
	assert.NoError(t, Validate(c))
}

func TestValidate_EnrichmentRejections(t *testing.T) {
	cases := map[string]func(*EnrichmentConfig){
		"duplicate provider": func(e *EnrichmentConfig) { e.Providers = append(e.Providers, e.Providers[0]) },
		"bad type":           func(e *EnrichmentConfig) { e.Providers[0].Type = "http" },
		"bad format":         func(e *EnrichmentConfig) { e.Providers[0].Format = "tsv" },
		"empty command":      func(e *EnrichmentConfig) { e.Providers[0].Command = nil },
		"empty key":          func(e *EnrichmentConfig) { e.Providers[0].Key = "" },
		"zero interval":      func(e *EnrichmentConfig) { e.Providers[0].RefreshIntervalSec = 0 },
		"zero timeout":       func(e *EnrichmentConfig) { e.Providers[0].TimeoutSec = 0 },
		"unknown col prov":   func(e *EnrichmentConfig) { e.Columns[0].Provider = "nope" },
		"empty field":        func(e *EnrichmentConfig) { e.Columns[0].Field = "" },
		"bad joinOn":         func(e *EnrichmentConfig) { e.Columns[0].JoinOn = "label" },
		"empty header":       func(e *EnrichmentConfig) { e.Columns[0].Header = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := Defaults()
			e := validEnrichment()
			mutate(&e)
			c.Enrichment = e
			assert.Error(t, Validate(c))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestValidate_Enrichment`
Expected: FAIL — `EnrichmentConfig`/`EnrichProvider`/`EnrichColumn` undefined.

- [ ] **Step 3: Add the config structs**

In `internal/config/config.go`, add the field to the `Config` struct (after the `Web` line):

```go
	Web        WebConfig        `yaml:"web"        json:"web"`
	Enrichment EnrichmentConfig `yaml:"enrichment,omitempty" json:"enrichment,omitempty"`
```

Then add the new types (place them after the `WebConfig` type definition):

```go
// EnrichmentConfig configures extra cluster-list columns joined from providers.
type EnrichmentConfig struct {
	Providers []EnrichProvider `yaml:"providers,omitempty" json:"providers,omitempty"`
	Columns   []EnrichColumn   `yaml:"columns,omitempty"   json:"columns,omitempty"`
}

// EnrichProvider is one enrichment data source. Only type "shell" is supported.
type EnrichProvider struct {
	Name               string   `yaml:"name"               json:"name"`
	Type               string   `yaml:"type"               json:"type"`
	Command            []string `yaml:"command,omitempty"  json:"command,omitempty"`
	Format             string   `yaml:"format,omitempty"   json:"format,omitempty"`
	Key                string   `yaml:"key,omitempty"      json:"key,omitempty"`
	RefreshIntervalSec int      `yaml:"refreshIntervalSec,omitempty" json:"refreshIntervalSec,omitempty"`
	TimeoutSec         int      `yaml:"timeoutSec,omitempty" json:"timeoutSec,omitempty"`
}

// EnrichColumn is one configured column reading a field from a provider row.
type EnrichColumn struct {
	Header   string `yaml:"header"          json:"header"`
	Provider string `yaml:"provider"        json:"provider"`
	Field    string `yaml:"field"           json:"field"`
	JoinOn   string `yaml:"joinOn"          json:"joinOn"`
	Width    int    `yaml:"width,omitempty" json:"width,omitempty"`
}
```

- [ ] **Step 4: Add the validator**

In `internal/config/validate.go`, add the function (near the other `validateX` functions):

```go
func validateEnrichment(e EnrichmentConfig) []error {
	var errs []error
	names := make(map[string]bool)
	for _, p := range e.Providers {
		if p.Name == "" {
			errs = append(errs, fmt.Errorf("config: enrichment provider has empty name"))
			continue
		}
		if names[p.Name] {
			errs = append(errs, fmt.Errorf("config: duplicate enrichment provider %q", p.Name))
		}
		names[p.Name] = true
		if p.Type != "shell" {
			errs = append(errs, fmt.Errorf("config: enrichment provider %q: unsupported type %q (want shell)", p.Name, p.Type))
		}
		if p.Format != "" && p.Format != "json" {
			errs = append(errs, fmt.Errorf("config: enrichment provider %q: unsupported format %q (want json)", p.Name, p.Format))
		}
		if len(p.Command) == 0 {
			errs = append(errs, fmt.Errorf("config: enrichment provider %q: command is empty", p.Name))
		}
		if p.Key == "" {
			errs = append(errs, fmt.Errorf("config: enrichment provider %q: key is empty", p.Name))
		}
		if p.RefreshIntervalSec <= 0 {
			errs = append(errs, fmt.Errorf("config: enrichment provider %q: refreshIntervalSec must be > 0", p.Name))
		}
		if p.TimeoutSec <= 0 {
			errs = append(errs, fmt.Errorf("config: enrichment provider %q: timeoutSec must be > 0", p.Name))
		}
	}
	for i, c := range e.Columns {
		if c.Provider == "" || !names[c.Provider] {
			errs = append(errs, fmt.Errorf("config: enrichment column %d references unknown provider %q", i, c.Provider))
		}
		if c.Field == "" {
			errs = append(errs, fmt.Errorf("config: enrichment column %d has empty field", i))
		}
		if c.JoinOn != "name" && c.JoinOn != "namespace" {
			errs = append(errs, fmt.Errorf("config: enrichment column %d: joinOn must be name or namespace", i))
		}
		if c.Header == "" {
			errs = append(errs, fmt.Errorf("config: enrichment column %d has empty header", i))
		}
	}
	return errs
}
```

Then wire it into `Validate` — add this line alongside the other `all = append(...)` calls:

```go
	all = append(all, validateEnrichment(c.Enrichment)...)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestValidate`
Expected: PASS (existing validate tests plus the new enrichment ones).

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): enrichment provider/column schema + validation"
```

---

### Task 5: Config → enrich builder

**Files:**
- Create: `cmd/kapish/enrichment.go`
- Test: `cmd/kapish/enrichment_test.go`

**Interfaces:**
- Consumes: `kconfig.EnrichmentConfig` (Task 4); `enrich.ProviderSpec`, `enrich.Column`, `enrich.NewShellProvider`, `enrich.NewManager` (Tasks 1–3).
- Produces:
  - `func buildEnrichmentManager(cfg kconfig.EnrichmentConfig) *enrich.Manager` — returns `nil` when no valid providers+columns exist; logs (`slog.Warn`) and skips malformed entries.

- [ ] **Step 1: Write the failing test**

Create `cmd/kapish/enrichment_test.go`:

```go
package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/enrich"
)

func TestBuildEnrichmentManager_Nil_WhenEmpty(t *testing.T) {
	assert.Nil(t, buildEnrichmentManager(kconfig.EnrichmentConfig{}))
}

func TestBuildEnrichmentManager_SkipsBadProvider(t *testing.T) {
	cfg := kconfig.EnrichmentConfig{
		Providers: []kconfig.EnrichProvider{
			{Name: "bad", Type: "http", Command: []string{"x"}, Key: "id", RefreshIntervalSec: 1, TimeoutSec: 1},
		},
		Columns: []kconfig.EnrichColumn{{Header: "X", Provider: "bad", Field: "f", JoinOn: "name"}},
	}
	// The only provider is invalid, so no columns survive -> nil manager.
	assert.Nil(t, buildEnrichmentManager(cfg))
}

func TestBuildEnrichmentManager_BuildsColumns(t *testing.T) {
	cfg := kconfig.EnrichmentConfig{
		Providers: []kconfig.EnrichProvider{
			{Name: "cost", Type: "shell", Command: []string{"echo", "[]"}, Key: "id", RefreshIntervalSec: 300, TimeoutSec: 5},
		},
		Columns: []kconfig.EnrichColumn{{Header: "COST", Provider: "cost", Field: "c", JoinOn: "name"}},
	}
	mgr := buildEnrichmentManager(cfg)
	require.NotNil(t, mgr)
	cols := mgr.Columns()
	require.Len(t, cols, 1)
	assert.Equal(t, "COST", cols[0].Header)

	// Cells resolvable (pending before any fetch).
	cells := mgr.Cells(enrich.ClusterRef{Name: "c1"})
	require.Len(t, cells, 1)
	assert.Equal(t, enrich.CellPending, cells[0].State)
	_ = context.Background()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/kapish/ -run TestBuildEnrichmentManager`
Expected: FAIL — `buildEnrichmentManager` undefined.

- [ ] **Step 3: Implement the builder**

Create `cmd/kapish/enrichment.go`:

```go
package main

import (
	"log/slog"
	"time"

	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/enrich"
)

// buildEnrichmentManager maps enrichment config to an enrich.Manager. It is
// defensive: a provider or column it can't build is logged and skipped so a bad
// block never breaks the cluster list. Returns nil when nothing usable remains.
func buildEnrichmentManager(cfg kconfig.EnrichmentConfig) *enrich.Manager {
	valid := make(map[string]bool)
	var specs []enrich.ProviderSpec
	for _, p := range cfg.Providers {
		if p.Type != "shell" || len(p.Command) == 0 || p.Key == "" || p.RefreshIntervalSec <= 0 || p.TimeoutSec <= 0 {
			slog.Warn("enrichment: skipping invalid provider", "name", p.Name, "type", p.Type)
			continue
		}
		specs = append(specs, enrich.ProviderSpec{
			Provider: enrich.NewShellProvider(p.Name, p.Command, p.Key),
			Interval: time.Duration(p.RefreshIntervalSec) * time.Second,
			Timeout:  time.Duration(p.TimeoutSec) * time.Second,
		})
		valid[p.Name] = true
	}

	var cols []enrich.Column
	for _, c := range cfg.Columns {
		if !valid[c.Provider] || c.Field == "" || (c.JoinOn != "name" && c.JoinOn != "namespace") {
			slog.Warn("enrichment: skipping invalid column", "header", c.Header, "provider", c.Provider)
			continue
		}
		cols = append(cols, enrich.Column{
			Header:   c.Header,
			Provider: c.Provider,
			Field:    c.Field,
			JoinOn:   c.JoinOn,
			Width:    c.Width,
		})
	}

	if len(specs) == 0 || len(cols) == 0 {
		return nil
	}
	return enrich.NewManager(specs, cols)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/kapish/ -run TestBuildEnrichmentManager`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add cmd/kapish/enrichment.go cmd/kapish/enrichment_test.go
git commit -m "feat(cmd): build enrich.Manager from config (log-and-skip bad blocks)"
```

---

### Task 6: TUI wiring — manager, subscription, message

**Files:**
- Modify: `internal/tui/model.go` (add `Enrichment` to `Config`; batch subscription in `Init`; handle `enrichUpdatedMsg` in `Update`)
- Modify: `internal/tui/messages.go` (add `enrichUpdatedMsg`)
- Modify: `internal/tui/commands.go` (add `enrichUpdatesCmd`)
- Test: `internal/tui/enrich_wiring_test.go`

**Interfaces:**
- Consumes: `enrich.Manager`, `enrich.ClusterRef`, `enrich.CellReady` (Tasks 1–2).
- Produces:
  - `tui.Config` gains `Enrichment *enrich.Manager` (nil-safe).
  - `type enrichUpdatedMsg struct{}`
  - `func (m Model) enrichUpdatesCmd() tea.Cmd` — returns `nil` when `Enrichment` is nil; otherwise blocks on `Updates()` and emits `enrichUpdatedMsg`, and is re-issued on each `enrichUpdatedMsg` to keep listening.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/enrich_wiring_test.go`:

```go
package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/v4run/kapish/internal/enrich"
)

func readyManager() *enrich.Manager {
	specs := []enrich.ProviderSpec{{
		Provider: enrich.NewStaticProvider("cost", map[string]enrich.Row{
			"prod-eu-1": {"monthlyCost": "$412"},
		}),
		Interval: time.Hour, Timeout: time.Second,
	}}
	cols := []enrich.Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	m := enrich.NewManager(specs, cols)
	m.RefreshAll(context.Background())
	return m
}

func TestUpdate_EnrichUpdatedMsgReRenders(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu", Enrichment: readyManager()})
	// enrichUpdatedMsg must be handled and return a command to keep listening.
	mu, cmd := m.Update(enrichUpdatedMsg{})
	_ = mu.(Model)
	assert.NotNil(t, cmd, "should re-subscribe after an enrich update")
}

func TestEnrichUpdatesCmd_NilWithoutManager(t *testing.T) {
	m := New(Config{})
	assert.Nil(t, m.enrichUpdatesCmd())
}

var _ tea.Msg = enrichUpdatedMsg{}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'TestUpdate_EnrichUpdated|TestEnrichUpdatesCmd'`
Expected: FAIL — `enrichUpdatedMsg`, `enrichUpdatesCmd`, and `Config.Enrichment` undefined.

- [ ] **Step 3: Add the config field**

In `internal/tui/model.go`, add the import and the field. Add to the import block:

```go
	"github.com/v4run/kapish/internal/enrich"
```

Add to the `Config` struct (after `ConfigPath`):

```go
	// Enrichment supplies extra cluster-list columns. May be nil.
	Enrichment *enrich.Manager
```

- [ ] **Step 4: Add the message type**

In `internal/tui/messages.go`, add:

```go
// enrichUpdatedMsg fires when the enrichment manager's cache changes.
type enrichUpdatedMsg struct{}
```

- [ ] **Step 5: Add the subscription command**

In `internal/tui/commands.go`, add:

```go
// enrichUpdatesCmd blocks on the enrichment manager's update channel and emits
// enrichUpdatedMsg. Returns nil when no manager is configured.
func (m Model) enrichUpdatesCmd() tea.Cmd {
	mgr := m.cfg.Enrichment
	if mgr == nil {
		return nil
	}
	return func() tea.Msg {
		<-mgr.Updates()
		return enrichUpdatedMsg{}
	}
}
```

- [ ] **Step 6: Wire Init and Update**

In `internal/tui/model.go`, change `Init` to also subscribe:

```go
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), m.refreshTickCmd(), m.enrichUpdatesCmd())
}
```

Add a case to the `Update` switch (place it near `refreshTickMsg`):

```go
	case enrichUpdatedMsg:
		// Cache changed; re-render happens automatically. Re-subscribe.
		return m, m.enrichUpdatesCmd()
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestUpdate_EnrichUpdated|TestEnrichUpdatesCmd'`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/model.go internal/tui/messages.go internal/tui/commands.go internal/tui/enrich_wiring_test.go
git commit -m "feat(tui): wire enrichment manager, subscription, update message"
```

---

### Task 7: TUI rendering — enrichment columns

**Files:**
- Modify: `internal/tui/style.go` (add `enrichCellRender` + `enrichCellText`)
- Modify: `internal/tui/view.go` (render enrichment columns + selected-row error line)
- Test: `internal/tui/view_test.go` (append)

**Interfaces:**
- Consumes: `enrich.Manager` via `m.cfg.Enrichment`; `enrich.ClusterRef`, `enrich.Cell`, `enrich.CellReady/Pending/Error/Empty`; existing `padRight`, `displayWidth`, `maxInt`, `styleDim`, `styleRed`.
- Produces:
  - `func enrichCellText(c enrich.Cell) string` — the plain text used for width and Ready rendering (`Ready`→text, `Pending`→`…`, `Error`→`!`, `Empty`→`-`).
  - `func enrichCellRender(c enrich.Cell, w int) string` — padded + styled cell.

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/view_test.go`:

```go
func TestView_EnrichmentColumns(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu", Enrichment: readyManager()})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = mu.(Model)
	mu, _ = m.Update(clustersLoadedMsg{clusters: []capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned", K8sVersion: "v1.30.2", Provider: "aws"},
		{Name: "other", Namespace: "prod", Phase: "Failed", Provider: "gcp"},
	}})
	m = mu.(Model)

	out := m.View()
	assert.Contains(t, out, "COST", "enrichment header shows")
	assert.Contains(t, out, "$412", "joined value shows for matching cluster")
	// "other" has no matching row -> empty marker rendered somewhere.
	assert.Contains(t, out, "-")
}

func TestView_EnrichmentPending(t *testing.T) {
	// Manager with a provider that has never fetched -> pending marker.
	specs := []enrich.ProviderSpec{{
		Provider: enrich.NewStaticProvider("cost", map[string]enrich.Row{"prod-eu-1": {"monthlyCost": "$1"}}),
		Interval: time.Hour, Timeout: time.Second,
	}}
	cols := []enrich.Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	mgr := enrich.NewManager(specs, cols) // NOT refreshed

	m := New(Config{MgmtContext: "mgmt-eu", Enrichment: mgr})
	mu, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m = mu.(Model)
	mu, _ = m.Update(clustersLoadedMsg{clusters: []capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned"},
	}})
	m = mu.(Model)
	assert.Contains(t, m.View(), "…", "pending marker shows before first fetch")
}
```

Add the imports this test needs to the top of `view_test.go` (if not already present): `"time"` and `"github.com/v4run/kapish/internal/enrich"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestView_Enrichment`
Expected: FAIL — enrichment columns not rendered; `enrichCellText` undefined.

- [ ] **Step 3: Add the cell helpers**

In `internal/tui/style.go`, add (and add `"github.com/v4run/kapish/internal/enrich"` to its imports):

```go
// enrichCellText is the plain (unstyled) text an enrichment cell occupies —
// used both for width measurement and for the Ready rendering.
func enrichCellText(c enrich.Cell) string {
	switch c.State {
	case enrich.CellReady:
		return c.Text
	case enrich.CellError:
		return "!"
	case enrich.CellEmpty:
		return "-"
	default: // CellPending
		return "…"
	}
}

// enrichCellRender pads the cell to width w and applies its state styling.
func enrichCellRender(c enrich.Cell, w int) string {
	text := padRight(enrichCellText(c), w)
	switch c.State {
	case enrich.CellReady:
		return text
	case enrich.CellError:
		return styleRed.Render(text)
	default: // Empty, Pending
		return styleDim.Render(text)
	}
}
```

- [ ] **Step 4: Render enrichment columns in viewList**

In `internal/tui/view.go`, add `"github.com/v4run/kapish/internal/enrich"` to the imports.

Replace the list-branch body (the `else` block that builds the header and row loop) with the version below. This keeps the existing built-in columns and appends enrichment columns before the trailing glyph; AGE is now padded to `ageW` so the enrichment columns line up.

```go
	} else {
		// Enrichment columns (may be none).
		var ecols []enrich.Column
		if mgr := m.cfg.Enrichment; mgr != nil {
			ecols = mgr.Columns()
		}

		// Precompute per-row enrichment cells and column widths.
		ecW := make([]int, len(ecols))
		for i, col := range ecols {
			ecW[i] = maxInt(displayWidth(col.Header), col.Width)
		}
		rowCells := make([][]enrich.Cell, len(m.filtered))
		ageW := displayWidth("AGE")
		now := time.Now()
		for r, c := range m.filtered {
			ageW = maxInt(ageW, displayWidth(ageString(c.CreationTimestamp, now)))
			if mgr := m.cfg.Enrichment; mgr != nil {
				cells := mgr.Cells(enrich.ClusterRef{Name: c.Name, Namespace: c.Namespace})
				rowCells[r] = cells
				for i := range ecols {
					ecW[i] = maxInt(ecW[i], displayWidth(enrichCellText(cells[i])))
				}
			}
		}

		// Dynamic name/ns widths (unchanged behavior).
		nameW := displayWidth("NAME")
		nsW := displayWidth("NS")
		for _, c := range m.filtered {
			nameW = maxInt(nameW, displayWidth(c.Name))
			nsW = maxInt(nsW, displayWidth(c.Namespace))
		}

		// Header.
		header := fmt.Sprintf("  %s %s %-14s %-9s %-10s %s",
			padRight("NAME", nameW), padRight("NS", nsW), "PHASE", "VERSION", "PROVIDER", padRight("AGE", ageW))
		for i, col := range ecols {
			header += " " + padRight(col.Header, ecW[i])
		}
		b.WriteString(styleDim.Render(header) + "\n")

		// Rows.
		for i, c := range m.filtered {
			cursor := "  "
			name := padRight(c.Name, nameW)
			if i == m.cursor {
				cursor = styleSelect.Render("▸ ")
				name = styleSelect.Render(name)
			}
			ver := c.K8sVersion
			if ver == "" {
				ver = "-"
			}
			prov := c.Provider
			if prov == "" {
				prov = "-"
			}
			line := fmt.Sprintf("%s%s %s %s %s %s %s",
				cursor, name, padRight(c.Namespace, nsW), phaseCell(c.Phase, 14),
				padRight(ver, 9), padRight(prov, 10), padRight(ageString(c.CreationTimestamp, now), ageW))
			for j := range ecols {
				line += " " + enrichCellRender(rowCells[i][j], ecW[j])
			}
			line += " " + phaseGlyph(c.Phase)
			b.WriteString(line + "\n")
		}

		// Explain the selected row's enrichment errors, if any.
		if mgr := m.cfg.Enrichment; mgr != nil && m.cursor >= 0 && m.cursor < len(rowCells) {
			perr := mgr.ProviderErrors()
			for j, col := range ecols {
				if rowCells[m.cursor][j].State == enrich.CellError {
					if e := perr[col.Provider]; e != nil {
						b.WriteString(styleRed.Render("  "+col.Header+": "+e.Error()) + "\n")
						break
					}
				}
			}
		}
	}
```

Note: this replaces the block that previously declared `now := time.Now()` inside it, so `now` is still declared exactly once here. Ensure the old list-branch body (from `b.WriteString(styleDim.Render(fmt.Sprintf("  %s %s %-14s ...`) through the end of the row loop) is fully removed.

- [ ] **Step 5: Run the TUI tests to verify they pass**

Run: `go test ./internal/tui/`
Expected: PASS — new enrichment tests plus all existing view/model tests (alignment, empty state, etc.).

- [ ] **Step 6: Verify alignment by eye with a UUID + enrichment**

Run:
```bash
go test ./internal/tui/ -run TestView_EnrichmentColumns -v
```
Expected: PASS. (Optional manual check: the built-in columns still line up under their headers with enrichment columns appended.)

- [ ] **Step 7: Commit**

```bash
git add internal/tui/style.go internal/tui/view.go internal/tui/view_test.go
git commit -m "feat(tui): render enrichment columns with pending/error/empty states"
```

---

### Task 8: Start the manager at app entry + example config/docs

**Files:**
- Modify: `cmd/kapish/tui.go` (build + start manager; pass into `tui.Config`)
- Modify: `internal/config/template.go` (document the `enrichment` block in the config template comment)

**Interfaces:**
- Consumes: `buildEnrichmentManager` (Task 5); `tui.Config.Enrichment` (Task 6); `enrich.Manager.Start` (Task 2).
- Produces: a running, subscribed enrichment manager in the real TUI.

- [ ] **Step 1: Wire the manager into runTUI**

In `cmd/kapish/tui.go`, add `"context"` and `"github.com/v4run/kapish/internal/enrich"` to the imports if not present. Just before `model := tui.New(tui.Config{`, insert:

```go
	// Build + start enrichment (nil when unconfigured or all blocks invalid).
	var enrichMgr *enrich.Manager
	if enrichMgr = buildEnrichmentManager(app.Enrichment); enrichMgr != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		enrichMgr.Start(ctx)
	}
```

Then add the field to the `tui.Config` literal:

```go
	model := tui.New(tui.Config{
		CapiClient:  client,
		AppConfig:   app,
		MgmtContext: client.Context(),
		OneShot:     app.UI.OneShot,
		ConfigPath:  cfgPath,
		Enrichment:  enrichMgr,
	})
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 3: Document the config block**

In `internal/config/template.go`, add commented documentation for the enrichment block near the other sections of the template string (keep it commented so the default config stays inert):

```go
# enrichment:
#   providers:
#     - name: cost              # unique id referenced by columns
#       type: shell             # only "shell" is supported
#       command: ["kapish-cost", "--all"]  # argv; run directly, never via a shell
#       format: json            # provider must print a JSON array of flat objects
#       key: clusterId          # field in each row used as the join value
#       refreshIntervalSec: 300
#       timeoutSec: 15
#   columns:
#     - header: COST/mo         # column header
#       provider: cost          # must reference a provider above
#       field: monthlyCost      # row field to display
#       joinOn: name            # cluster field matched against the provider key (name|namespace)
#       width: 10               # optional; 0 = auto-size
```

- [ ] **Step 4: Run the full suite**

Run: `go build ./... && go test ./...`
Expected: PASS across all packages.

- [ ] **Step 5: Commit**

```bash
git add cmd/kapish/tui.go internal/config/template.go
git commit -m "feat: start enrichment manager at entry; document config block"
```

---

## Self-Review

**Spec coverage:**
- Provider interface + batch fetch → Task 1 (`Provider`, `Fetch`).
- Manager cache, tickers, `Cells()`, `Updates()` → Tasks 1–2.
- Async/non-blocking/cached, graceful degradation → Task 2 (`Start`, immediate + interval fetch; error keeps last-good rows) + Task 7 (state markers).
- Join model (`joinOn` → provider key → field) → Task 1 `Cells`.
- `capi.Cluster` untouched → confirmed: no task modifies `capi`; join uses `ClusterRef`.
- Config schema + validation → Task 4; runtime log-and-skip → Task 5.
- Shell contract (argv-only, JSON array, batch-keyed, timeout, missing/dup key skip) → Task 3.
- TUI rendering (columns after built-ins, `…`/`!`/`-`, auto-size, selected-row error line) → Task 7.
- Wiring/subscription → Tasks 6 & 8.
- Docs/example config → Task 8.
- Web columns / HTTP/labels/in-cluster providers / TSV / label joins → explicitly out of scope (spec non-goals); no tasks, by design.

**Placeholder scan:** No TBD/TODO; every code step shows full code; every test step shows assertions and a run command with expected result.

**Type consistency:** `Manager`, `ProviderSpec`, `Column`, `Cell`, `CellState` consts, `ClusterRef`, `NewShellProvider(name, argv, keyField)`, `NewStaticProvider(name, rows)`, `Cells() []Cell`, `Columns() []Column`, `Updates() <-chan struct{}`, `ProviderErrors() map[string]error`, `Start(ctx)`, `RefreshAll(ctx)`, `refreshOnce(ctx, name)` are used consistently across tasks. `buildEnrichmentManager` returns `*enrich.Manager` and is consumed in Task 8. `tui.Config.Enrichment *enrich.Manager`, `enrichUpdatedMsg`, `enrichUpdatesCmd`, `enrichCellText`, `enrichCellRender` names match between definition and use.
