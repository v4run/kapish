# Web UI settings fixes — design

Date: 2026-05-12

## Problem

Three issues in the web UI settings page (`internal/web/frontend/src/SettingsView.tsx`):

1. Clicking items in the left nav only changes the right-panel title — the same full form
   (all sections) is shown regardless of selection. The tabs imply section-scoped views.
2. Settings are persisted on every keystroke (`applyForm` → `save` on each `onChange`).
   There should be an explicit Save button.
3. The working-directory input is passed through verbatim; `~` and `$HOME` are not expanded,
   so `~/work` becomes a literal directory name.

(Separately: the "management cluster selector" question was a documentation question, answered
in conversation — it switches which CAPI management cluster kapish queries for its workload-cluster
inventory. No code change.)

## Changes

### 1. Section-scoped settings panel with per-section Save

- `SettingsSectionForm` gains a `section` prop (`'shell' | 'env' | 'aliases' | 'cwd'`) and renders
  only that section's fields:
  - `shell`: Shell select + Prompt template
  - `env`: Environment variables `KVList`
  - `aliases`: Aliases `KVList`
  - `cwd`: Working directory `TextField`
- Each section component holds **local draft state**, seeded from the loaded `KapishConfig`.
  - A `Save` button, disabled until the draft differs from the saved config.
  - A `Discard` button, resets the draft to the saved config.
  - The `saving… / saved / error` indicator sits next to that section's Save button.
- `SettingsView` responsibilities shrink: load `cfg` via `getConfig()`, render `AppHeader` +
  `SettingsTabs`, render the active section, expose a `save(nextCfg)` callback that PUTs and on
  success updates `cfg` (so other sections re-seed from the latest value). The single `val` /
  `applyForm` state is removed.
- Save merges the section's slice into the **latest** `cfg` before PUT (so concurrent edits in
  another section that were already saved are not clobbered). Switching tabs with unsaved changes
  keeps the draft in component state only if the component stays mounted; simplest acceptable
  behavior: drafts live in `SettingsView` keyed by section so they survive tab switches. If that
  proves fiddly, falling back to per-component state (draft lost on tab switch) is acceptable for v1.
- Theme: keep instant-apply (live preview). No Save button in the Theme tab. `toggleTheme` stays.
- Update the working-directory hint to: `~ and $HOME are expanded; empty = inherit`.

### 2. `~` / `$HOME` expansion in working directory (backend, at shell-spawn time)

- New helper in `internal/shell` (e.g. `expand.go`):
  ```go
  // expandCwd expands a leading ~ (or ~/) to the user's home dir, then expands
  // $VAR / ${VAR} via the process environment. A bare ~user form is returned unchanged.
  func expandCwd(p string) string
  ```
  - `""` → `""`.
  - `~` → home; `~/x` → `<home>/x`. Uses `os.UserHomeDir()`; on error, leaves the `~` as-is.
  - Then `os.ExpandEnv` on the result.
- Replace the duplicated `if opts.Cwd != "" { b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n") }`
  blocks in `zsh.go`, `bash.go`, `fish.go` with a shared helper:
  ```go
  // cdLine returns a "cd <quoted-expanded-cwd>\n" line, or "" when cwd is empty.
  func cdLine(cwd string) string
  ```
  fish uses the same `cd 'x'` syntax already, so one helper covers all three.
- Config (`shell.cwd`) continues to store the literal value (e.g. `~/work`) — portable across hosts.
- The TUI's config viewer (`internal/tui/view.go`) still shows the literal value; no change.

### 3. Management cluster selector

No code change. (Optional follow-up not in scope: a tooltip on the header chip.)

### 4. "Select a cluster" empty state

In `App.tsx`, when no cluster is selected the empty state is rendered as
`<div className="flex-1 flex"><SelectClusterEmpty /></div>`. `SelectClusterEmpty` → `EmptyState`
has a root of `h-full flex flex-col items-center justify-center ... text-center`, but with no width
it shrinks to content width as a flex item, so it sits flush-left in the pane.

- Fix: give the empty state full width of the pane. Change the wrapper to `flex-1` (drop the inner
  `flex`) so `EmptyState`'s `h-full` + the wrapper's `flex-1` width make it fill the pane and the
  existing `items-center` / `text-center` take effect. (`NoClustersFoundEmpty` in the sidebar is
  unaffected — it lives in a fixed-width column.)
- Replace the bespoke inline SVG icon in `SelectClusterEmpty` with the main logo:
  `icon={<KapishMark size={36} />}` (drop the `text-muted` wrapper styling so the logo keeps its
  accent/violet colors). Import `KapishMark` from `../brand/KapishMark`.

### 5. CAPI Cluster API: v1beta1 → v1beta2

`internal/capi` imports `sigs.k8s.io/cluster-api/api/core/v1beta1`; recent CAPI management
clusters log "v1beta1 Cluster is deprecated, use v1beta2 Cluster". `cluster-api v1.13.1` (already
in `go.mod`) ships `api/core/v1beta2`, which is the storage version. Migrate the package:

- Swap the import alias `clusterv1` to `sigs.k8s.io/cluster-api/api/core/v1beta2` in `list.go`,
  `watch.go`, `types.go` (and the three `_test.go` files).
- `list.go`: change `clusterGVR.Version` from `v1beta1` to `v1beta2`; update the doc comment.
- `types.go` — `FromV1Beta1` → rename to `FromV1Beta2` (update callers in `list.go`/`watch.go`),
  and adjust field access for the v1beta2 shape:
  - `v.Status.Phase` — unchanged.
  - `v.Status.ControlPlaneReady` → `v.Status.Initialization.ControlPlaneInitialized` (`*bool`;
    nil ⇒ false). Keep kapish's field name `ControlPlaneReady` (it's our own view type).
  - `v.Status.InfrastructureReady` → `v.Status.Initialization.InfrastructureProvisioned` (`*bool`;
    nil ⇒ false). Keep kapish's field name `InfrastructureReady`.
  - `v.Spec.Topology` is now a value (not a pointer): read `v.Spec.Topology.Version` directly
    (empty string when the topology is not defined) instead of the `!= nil` guard.
  - `v.Spec.InfrastructureRef` is now a value `ContractVersionedObjectReference` (not a pointer):
    guard on `v.Spec.InfrastructureRef.Kind != ""` (or `.IsDefined()`) then `providerFromKind(...)`.
  - `providerFromKind` is unchanged.
- Update the package/type doc comments that say "v1beta1".
- Update `internal/capi/*_test.go` to construct `v1beta2.Cluster` fixtures with the new
  `Status.Initialization` / value-typed `Spec.Topology` / `Spec.InfrastructureRef` shapes, and call
  `FromV1Beta2`.
- No change to consumers (`internal/tui`, `internal/web`) — they only touch kapish's `Cluster` type.

Note: this drops the ability to read clusters from a management cluster that *only* serves
`v1beta1` (pre-CAPI-v1.11). That's acceptable — those versions are EOL and the deprecation warning
only appears on versions that already serve v1beta2.

### 6. CI workflow validation

Add automated validation that the GitHub Actions workflows (`.github/workflows/*.yml`) are
well-formed, so a typo in YAML or an unknown action input is caught in CI rather than on push.

- Add an `actionlint` step to `ci.yml` (a `lint` job, or a step in the existing `test` job):
  ```yaml
  - name: Lint workflows
    uses: raven-actions/actionlint@v2
  ```
  `actionlint` validates workflow syntax, expression syntax, `runs-on` labels, `uses:` refs, and
  shell snippets in `run:` blocks (via shellcheck).
- Add a `make lint-actions` target that runs actionlint locally via `go run`
  (`go run github.com/rhysd/actionlint/cmd/actionlint@latest`), and wire it into the existing
  `lint` target so `make lint` covers Go vet + workflow lint. (No new entry in `go.mod` — `go run`
  with a version suffix uses an ephemeral module cache.)
- This is the "test cases for the workflows": `actionlint` is the validator; a green
  `Lint workflows` step (and `make lint-actions` locally) is the pass criterion. It exercises both
  `build.yml` and `ci.yml`, including the matrix expansion and the `run:` scripts.

## Testing

- Go: unit test for `expandCwd` (empty, `~`, `~/x`, `$HOME/x`, `${HOME}/x`, plain absolute path,
  `~unknownuser` left unchanged) and a small test that `cdLine` quotes and returns `""` for empty.
- Frontend: no test harness in the repo; verify via `cd internal/web/frontend && npm run build` and
  manual click-through (each tab shows only its fields; Save disabled until edit; Discard restores;
  saved indicator appears; reload shows persisted values).
- Go: update `internal/capi/*_test.go` for v1beta2 fixtures; `go vet ./...` and `go build ./...`
  must pass (catches missed v1beta1 field references).
- Workflows: `make lint-actions` (actionlint) passes against `.github/workflows/*.yml`.
- Run `make test`, `make lint`, and `make frontend` before finishing.

## Out of scope

- Re-architecting the config API or adding partial-update endpoints (PUT remains whole-config).
- Validating that the working directory exists (the shell `cd` failing is surfaced in the terminal).
- pwsh/nu shells.
