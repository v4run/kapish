# Cluster List Enrichment — Design

Date: 2026-08-30
Status: Approved (design); implementation not started

## Summary

Add a way to enrich cluster-list rows with extra columns whose values are
joined in from sources outside the CAPI `Cluster` CR. The mechanism is a
pluggable **enrichment provider** abstraction: a provider fetches keyed rows on
its own schedule, and configured **columns** join those rows onto clusters at
render time. Fetching is asynchronous, cached, and never blocks or crashes the
cluster list.

Four provider types are envisioned — shell/CLI command, external HTTP API,
cluster labels/annotations, and related in-cluster resources. This spec covers
the shared architecture plus the **shell/CLI provider** as the first end-to-end
implementation (MVP). The other three plug into the same `Provider` interface in
later work.

## Goals

- A generic, config-driven column-enrichment pipeline that any provider type can
  feed.
- Async, non-blocking, cached fetching: the cluster list keeps rendering
  instantly from CAPI; enrichment cells fill in when ready.
- Graceful degradation: a slow or failing source shows a per-cell marker, never
  stalls or crashes the list.
- `capi.Cluster` stays untouched — enrichment is merged in the render layer.
- MVP renders in the TUI; the engine is UI-agnostic so the web UI follows later.

## Non-goals (this spec)

- HTTP, labels/annotations, and in-cluster-resource providers (future; same
  interface).
- Web UI column rendering (deferred to a follow-up; see "Phasing").
- TSV / non-JSON provider output (JSON only in MVP).
- Label/annotation-based join keys (MVP joins on `name`/`namespace` only).

## Architecture

### New package `internal/enrich`

Deliberately decoupled from `capi` (k8s-facing) and from the renderers
(`tui`/`web`). It imports neither. The manager is fed a lightweight snapshot of
clusters so it never depends on `capi`.

```
internal/enrich/
  types.go       # Column, Cell, CellState, Row, ClusterRef, Provider
  manager.go     # Manager: cache, per-provider tickers, Cells(), Updates()
  shell.go       # shellProvider (argv exec, JSON parse, timeout)
  *_test.go
```

### Core types

```go
// ClusterRef is the minimal cluster identity the manager needs. Avoids a
// dependency on the capi package.
type ClusterRef struct {
    Name      string
    Namespace string
}

// Row is one provider output record: field name -> stringified value.
type Row map[string]string

// Provider fetches all rows for one source, once per refresh, keyed by the
// provider's own key value.
type Provider interface {
    Name() string
    Fetch(ctx context.Context) (map[string]Row, error) // keyed by join value
}

// Column is one configured enrichment column.
type Column struct {
    Header   string // display header
    Provider string // provider name this column reads from
    Field    string // which field of the joined Row to display
    JoinOn   string // cluster field matched against the provider row key: name|namespace
    Width    int    // optional; 0 = auto-size like NAME/NS
}

type CellState int

const (
    CellReady   CellState = iota // Text is a joined value
    CellPending                  // provider has not fetched yet
    CellError                    // provider's last fetch failed (cache may be stale)
    CellEmpty                    // fetched OK but no matching row for this cluster
)

type Cell struct {
    Text  string
    State CellState
}
```

### Manager

`enrich.Manager` owns the providers and a mutex-guarded cache.

- Per provider, a ticker goroutine runs at that provider's `refreshIntervalSec`,
  calls `Fetch` with a `timeoutSec` context, and stores
  `{rows map[string]Row, fetchedAt time.Time, lastErr error}`.
- On `Fetch` error, the cache **keeps the last good rows** (if any) and records
  `lastErr`; cells derived from stale-but-present rows still render their text,
  and the manager exposes the error for surfacing (see Rendering). A provider
  that has never succeeded yields `CellPending` before first fetch, `CellError`
  after a failed first fetch.
- `Cells(c ClusterRef) map[string]Cell` — for each configured column, compute
  the join value `c.<JoinOn>`, look up `rows[joinValue]`, and return a `Cell`:
  - never fetched → `CellPending`
  - failed with no prior rows → `CellError`
  - fetched, no matching row → `CellEmpty`
  - matching row, `Field` present → `CellReady{Text}` (missing `Field` → empty text, `CellReady`)
  - The map key is a stable per-column id (column index or `provider/field`).
- `Updates() <-chan struct{}` — fires (non-blocking, coalesced) whenever any
  provider's cache changes, so subscribers re-render.
- `Start(ctx)` / lifecycle tied to the app context; goroutines stop on cancel.

### Join model

A column declares `provider`, `field` (row field to display), and `joinOn`
(cluster field matched against the provider's row key). The manager does the
lookup at render time. `capi.Cluster` is not modified; enrichment is merged into
the TUI model / web payload at the render boundary.

### Wiring

- **TUI**: `cmd/kapish/tui.go` constructs a `Manager` from config and passes it
  into the model. The model subscribes to `Updates()` via a bubbletea command
  (same shape as the existing cluster watch) that emits an `enrichUpdatedMsg`,
  triggering a re-render. `viewList` calls `Cells()` per visible row.
- **Web** (later phase): the web server shares one `Manager`, merges `Cells`
  into the per-cluster SSE payload, and pushes on `Updates()`.

## Configuration

New top-level `enrichment` block, following the existing `yaml`+`json` tag style
so the web config API keeps working.

```yaml
enrichment:
  providers:
    - name: cost                        # unique id, referenced by columns
      type: shell                       # only "shell" implemented in MVP
      command: ["kapish-cost", "--all"] # argv; NOT run through a shell
      format: json                      # json only in MVP
      key: clusterId                    # field in each output row = join value
      refreshIntervalSec: 300
      timeoutSec: 15
  columns:
    - header: COST/mo                   # column header text
      provider: cost                    # must reference a provider above
      field: monthlyCost                # which field of the joined row to display
      joinOn: name                      # cluster field matched against provider key
      width: 10                         # optional; 0 = auto-size
```

### Validation (`internal/config/validate.go`)

- provider `name`s are unique
- every column's `provider` references an existing provider
- provider `type` ∈ {`shell`}
- `format` ∈ {`json`}
- `refreshIntervalSec` > 0 and `timeoutSec` > 0
- `command` is non-empty
- `joinOn` ∈ {`name`, `namespace`}

`config validate` fails on an invalid enrichment block. At runtime, because
enrichment is additive, the app **logs and skips** a bad enrichment block rather
than refusing to start.

## Shell provider contract

- Runs `command` as argv directly via `exec.CommandContext` — **no `sh -c`**, so
  there is no shell-injection surface. The context carries `timeoutSec`.
- On timeout or non-zero exit → provider enters the error path (cache keeps last
  good rows if present; `lastErr` recorded).
- Expects stdout to be a **JSON array of flat objects**, e.g.
  `[{"clusterId":"a1b2…","monthlyCost":"$412","nodes":12}]`. Non-string values
  are stringified for display.
- The `key` field of each object supplies the join value. A row with a missing
  or duplicate key is skipped (logged once), not fatal.
- Runs **once per refresh** (batch), regardless of cluster count; many columns
  may reference the same provider and share its single fetch.

## Rendering

### TUI (`viewList`)

Enrichment columns render after the built-in columns and before the trailing
phase glyph:

```
NAME · NS · PHASE · VERSION · PROVIDER · AGE · <enrichment columns…> · glyph
```

Columns appear in config order. Each auto-sizes to `max(header, widest cell)`
using the same dynamic-width logic that fixed the UUID-name alignment, unless a
`width` hint is set. Cell states:

- `CellReady`  → the text
- `CellPending` → dim `…`
- `CellError`  → red `!`; the **selected** row's provider error message surfaces
  in a detail line above the status bar, so `!` is explainable without
  cluttering every row
- `CellEmpty`  → dim `-` (distinct from pending)

### Web

Deferred to a follow-up phase. The web list uses a fixed CSS grid
(`grid-cols-[1fr_auto_auto_auto]`), so dynamic columns require real layout work
(dynamic grid template, header row, per-cell state styling, SSE payload
extension). The `Manager` and shell provider are UI-agnostic, so nothing is
thrown away — the web columns are purely additive.

## Testing

- `manager_test.go` — fake `Provider`: join by `name`/`namespace`; `CellPending`
  before first fetch; `CellError` on failure while keeping last-good rows;
  `CellEmpty` for no-match; cache refresh on tick; concurrent `Fetch`/`Cells`
  under the race detector.
- `shell_test.go` — real `exec` of tiny scripts: happy-path JSON; timeout via
  `sleep`; non-zero exit; malformed JSON; missing/duplicate key rows.
- `view_test.go` — enrichment columns render in config order; `…`/`!`/`-`
  states; alignment holds with a UUID name plus enrichment columns.
- `validate_test.go` — a valid config plus each rejection rule; runtime
  log-and-skip of a bad enrichment block.

## Phasing / build order

Within this plan:

1. `types.go` — core types and the `Provider` interface.
2. `manager.go` — cache, tickers, `Cells()`, `Updates()` (TDD with a fake provider).
3. `shell.go` — shell provider (argv exec, JSON parse, timeout).
4. Config schema + validation.
5. TUI wiring + render (`enrichUpdatedMsg`, `viewList` columns, cell states).
6. Docs + example config.

Explicitly **out of scope** for this plan (tracked as follow-ups):

- Web UI column rendering.
- HTTP, labels/annotations, and in-cluster-resource providers.
- TSV output and label-based join keys.
