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

## Testing

- Go: unit test for `expandCwd` (empty, `~`, `~/x`, `$HOME/x`, `${HOME}/x`, plain absolute path,
  `~unknownuser` left unchanged) and a small test that `cdLine` quotes and returns `""` for empty.
- Frontend: no test harness in the repo; verify via `cd internal/web/frontend && npm run build` and
  manual click-through (each tab shows only its fields; Save disabled until edit; Discard restores;
  saved indicator appears; reload shows persisted values).
- Run `make test` and `make frontend` before finishing.

## Out of scope

- Re-architecting the config API or adding partial-update endpoints (PUT remains whole-config).
- Validating that the working directory exists (the shell `cd` failing is surfaced in the terminal).
- pwsh/nu shells.
