# kapish — design

**Date:** 2026-05-09
**Status:** Approved (pending implementation plan)

## Overview

kapish is a debugging tool for engineers operating Kubernetes clusters managed by [Cluster API (CAPI)](https://cluster-api.sigs.k8s.io/). It lists CAPI workload clusters from a management cluster and lets the user pick one to drop into a shell whose `KUBECONFIG` is pre-set to that cluster — plus configurable env vars, aliases, working directory, and prompt prefix. It runs in two modes: a TUI for terminal use and a localhost Web UI with an in-browser terminal.

It is **not** a cluster browser (k9s territory) and not a multi-user platform. Each user runs their own instance against their own kubeconfig.

## Goals

- Make hopping between CAPI clusters fast: pick from a live-updated list → land in a shell with the right kubeconfig.
- Same functionality available in TUI and Web UI from a single binary.
- Keep configuration simple: one global config file, optionally edited via in-app settings.
- Stay out of the way once the shell is open — kapish is a launcher, not a workbench.

## Non-goals (v1)

- In-UI resource browsing (pods, logs, etc.) — that's k9s.
- Multi-user / hosted deployments. No auth backend.
- Per-cluster or per-namespace config overrides — global only.
- Hooks (pre/post shell spawn), customizable TUI keybindings.
- Shells beyond bash, zsh, fish (nu and pwsh deferred to v2).
- `{git}` and `{region}` prompt tokens.

## Architecture

Single Go binary `kapish`, distributed as a static binary. Cobra subcommands:

| Subcommand | Behavior |
|---|---|
| `kapish` (no args) | TUI mode (bubbletea full-screen) |
| `kapish serve [--port 0\|N] [--no-open] [--bind 127.0.0.1] [--dev]` | Starts HTTP+WebSocket server, opens browser by default. `--dev` proxies to Vite for HMR. |
| `kapish version` | Print version |
| `kapish config edit` | Opens the YAML config file in `$EDITOR` |
| `kapish config validate` | Prints the merged effective config (after flags/env/file/defaults) |

### Global flags

`--kubeconfig`, `--context` (override mgmt cluster), `--config <path>` (override config file path), `--one-shot` (TUI exits after first shell exit instead of returning to list), `--log-level`, `--log-file`.

### Internal Go packages

| Package | Purpose |
|---|---|
| `internal/capi` | Connect to mgmt cluster, list/watch `Cluster` CRDs, fetch workload kubeconfig from the cluster's kubeconfig Secret, mgmt-cluster picker logic |
| `internal/config` | Load YAML, merge with flags + env + defaults, validate, expose typed config; in-place YAML edits via `yaml.v3` Node API (preserves comments + ordering) |
| `internal/shell` | Detect `$SHELL`, generate per-session init for bash/zsh/fish, spawn process, manage env+aliases+prompt+cwd, manage temp-kubeconfig lifecycle |
| `internal/tui` | bubbletea models for the cluster list, filter, settings screen, status bar |
| `internal/web` | HTTP handlers (cluster list JSON, SSE stream, session create, config GET/PUT), WebSocket-PTY bridge |
| `internal/web/frontend` | React + TypeScript + Tailwind + xterm.js SPA; build artifacts embedded via `//go:embed` |
| `cmd/kapish` | Cobra entry point |

### Key external dependencies

- `sigs.k8s.io/cluster-api/api/v1beta1` — CAPI types
- `k8s.io/client-go` — K8s API access
- `github.com/charmbracelet/bubbletea` + `bubbles` + `lipgloss` — TUI
- `github.com/creack/pty` — PTY allocation
- `github.com/coder/websocket` — WebSocket
- `github.com/spf13/cobra` — CLI subcommands
- `gopkg.in/yaml.v3` — YAML parsing with comment-preserving roundtrip

### Frontend dependencies

- `react` 18, `react-dom` — UI
- `xterm` + `xterm-addon-fit` — terminal
- `vite` — build
- `tailwindcss` — styling
- `@tanstack/react-query` — cluster list fetching
- No `react-router`, no UI component library (everything in-house, per design handoff)

## Configuration

### File location (resolved in order)

1. `--config <path>` flag
2. `$KAPISH_CONFIG` env var
3. `$XDG_CONFIG_HOME/kapish/config.yaml` (default `~/.config/kapish/config.yaml`)
4. Built-in defaults (no file required for first run)

### Precedence

Flag > env var > config file > built-in defaults. Within the management cluster, `--kubeconfig`/`--context` flags > config file > kubeconfig's current-context.

### Schema

```yaml
managementClusters:
  current: prod-mgmt          # name; must match an entry below (or be empty for first/single)
  entries:
    - name: prod-mgmt
      kubeconfig: ""          # optional; default: $KUBECONFIG or ~/.kube/config
      context: ""             # optional; default: kubeconfig's current-context
      namespace: ""           # optional; "" = all namespaces
    - name: stg-mgmt
      kubeconfig: ~/.kube/staging-mgmt
      context: stg-admin

# A single managementCluster: {kubeconfig, context, namespace} block is also accepted
# for backward-compat / simplicity; it's normalized into a one-entry list internally.

shell:
  command: ""                 # default: $SHELL ("/bin/bash" fallback)
                              # supported v1: bash, zsh, fish (basename match)
  cwd: ""                     # working directory for spawned shell; "" = inherit
  env:                        # injected env vars
    EDITOR: vim
  aliases:                    # injected as bash/zsh aliases or fish abbrs
    k: kubectl
    kgp: kubectl get pods -A
  prompt: "[{cluster}] "      # prepended to existing PS1; tokens: {cluster} {ns} {provider} {ctx} {time}

ui:
  theme: dark                 # dark | light
  refreshIntervalSec: 30      # how often to re-poll mgmt cluster's Cluster list
  oneShot: false              # if true, TUI exits after first shell exit

web:
  defaultPort: 0              # 0 = pick free port
  openBrowser: true           # auto-open browser on `kapish serve`
  bindAddr: "127.0.0.1"       # any non-loopback requires explicit --bind override
```

### Configuration UX (3 paths to the same source of truth)

Settings tabs (identical in TUI and Web UI in v1):
- **Shell & prompt** — shell command, prompt template, working dir
- **Env vars** — KV editor
- **Aliases** — KV editor
- **Working dir** — text input (also reachable from Shell & prompt; separated as a tab to match the design handoff)
- **Theme** — dark/light toggle

Hooks and Keybinds tabs from the design handoff are deferred to v2 — absent from v1 UI surface.

The mgmt-cluster picker is a header chip dropdown (Web UI) or a `m`-key overlay (TUI) — it switches `managementClusters.current` immediately. Adding/editing/removing `managementClusters.entries[]` is **YAML-only in v1** (use `kapish config edit`); a UI surface for entry management is v2.

1. **In-TUI settings** — press `s` from the cluster list. Form-based, navigated with arrow keys. Save writes back to the YAML.
2. **In-Web UI settings** — `/settings` route with the same tabs and same persistence.
3. **CLI escape hatches** — `kapish config edit` (opens YAML in `$EDITOR`), `kapish config validate` (prints merged effective config). No `config set` command — UI or `$EDITOR` covers it.

### Hot-reload semantics

| Setting | When applied |
|---|---|
| `shell.*` | Next shell spawn (already-running shells keep their original env) |
| `ui.theme` | Immediately |
| `ui.refreshIntervalSec`, `ui.oneShot` | Immediately |
| `web.openBrowser`, `web.bindAddr`, `web.defaultPort` | Next `kapish serve` invocation |
| `managementClusters.current` (picker switch in TUI/Web UI) | Immediately — rebuilds K8s client + watch. Live PTY sessions keep their already-fetched kubeconfig and continue running. |
| `managementClusters.entries[*]` (edited via YAML; v1 has no settings UI for entries) | Requires kapish restart. v2 will hot-reload entry adds/edits via the settings UI. |

### YAML round-tripping & concurrency

Read → mutate via `yaml.v3` `Node` tree → write whole-file under `flock`. User comments and key ordering preserved. Concurrent edits (TUI + Web UI both running) serialize on the lock.

### Validation

Blocks save with inline UI errors:
- `shell.command` resolves via `exec.LookPath`
- Env-var keys match `^[A-Z_][A-Z0-9_]*$`
- Alias names are valid shell identifiers
- `shell.prompt` parses (tokens: `{cluster} {ns} {provider} {ctx} {time}`; unknown tokens reported)
- Each entry in `managementClusters.entries[*]` has a unique `name`; `current` references an existing entry; the referenced kubeconfig+context resolves at validate time

### First-run experience

If no config file exists, `kapish` writes a commented-out template to `~/.config/kapish/config.yaml` so users see all keys and defaults. Boot still works without a file.

## Cluster discovery & kubeconfig fetching

### Listing

- Build a `client-go` config from the resolved kubeconfig+context for the current mgmt cluster, register the CAPI scheme.
- LIST `cluster.x-k8s.io/v1beta1` `Cluster` resources, then WATCH for incremental updates.
- Scope: all namespaces by default; `entries[*].namespace` narrows it.
- Fallback: periodic re-LIST every `ui.refreshIntervalSec` to catch missed watch events.
- The TUI/Web UI re-render reactively on add/update/delete events.

### Fields surfaced per cluster

| Field | Source |
|---|---|
| Name | `metadata.name` |
| Namespace | `metadata.namespace` |
| Phase | `status.phase` (`Pending`, `Provisioning`, `Provisioned`, `Deleting`, `Failed`) |
| ControlPlaneReady | `status.controlPlaneReady` |
| InfrastructureReady | `status.infrastructureReady` |
| K8s version | `spec.topology.version` if ClusterClass; else best-effort from referenced control-plane object |
| Age | derived from `metadata.creationTimestamp` |
| Provider | derived from `spec.infrastructureRef.kind` (e.g., `AWSCluster` → `aws`); known providers (aws, gcp, azure, vsphere, hetzner) get tinted chips, others fall back to neutral |

### Workload kubeconfig

CAPI convention: a Secret named `<cluster-name>-kubeconfig` in the cluster's namespace, with key `value` containing the raw kubeconfig YAML.

On cluster-select:
1. GET that Secret from the management cluster.
2. Decode `data.value` (`client-go` decodes base64 for us).
3. Write to a per-session temp file via `os.MkdirTemp(os.TempDir(), "kapish-*")` + `os.CreateTemp(dir, "*.kubeconfig")`, mode `0600`, dir mode `0700`.
4. Set `KUBECONFIG=<that path>` in the spawned shell's env.
5. On shell exit (TUI or Web): `defer os.RemoveAll(tempDir)`.
6. On kapish exit (signal/crash): cleanup pass removes any leftover `kapish-*` temp dirs older than the process start time.
7. On kapish startup: belt-and-suspenders sweep removes stale `kapish-*` temp dirs older than 24h.

Why a temp file (vs in-memory): standard kubectl/helm/most K8s tooling expect a file path. Temp file is the lingua franca and works without surprises. `0600` keeps it from other local users.

## Shell spawn mechanics

The init must (1) source the user's normal rc so existing aliases/functions still work, (2) inject kapish env+aliases, (3) set `KUBECONFIG`, (4) apply the cluster-scoped prompt prefix, (5) cd to `shell.cwd` if set.

### bash

Spawn with `--rcfile <kapish-init>`. Init file:

```sh
[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"
export KUBECONFIG="<temp-path>"
export FOO="bar"             # from config.shell.env
alias k='kubectl'            # from config.shell.aliases
PS1='[my-cluster] '"$PS1"    # rendered from config.shell.prompt
[ -n "<cwd>" ] && cd "<cwd>"
```

### zsh

Set `ZDOTDIR=<kapish-tempdir>`, write a `.zshrc` in that dir:

```sh
[ -f "$HOME/.zshrc" ] && . "$HOME/.zshrc"
export KUBECONFIG="<temp-path>"
export FOO="bar"
alias k='kubectl'
PROMPT='[my-cluster] '"$PROMPT"
[ -n "<cwd>" ] && cd "<cwd>"
```

We pass through any user-set `ZSH_CUSTOM` so frameworks (oh-my-zsh, etc.) keep working.

### fish

Spawn with `fish --init-command="<init-string>"`:

```fish
set -gx KUBECONFIG "<temp-path>"
set -gx FOO "bar"
alias k 'kubectl'
function fish_prompt
  echo -n '[my-cluster] '
  # call user's prompt if defined, else fallback
end
[ -n "<cwd>" ] && cd "<cwd>"
```

### Shell detection

`filepath.Base(shell.command)` → `bash`/`zsh`/`fish` → matching codepath. Unknown basenames fall back to bash-style with a warning. v1 explicitly does not support `nu` or `pwsh`.

### Prompt tokens

`{cluster}`, `{ns}` (namespace), `{provider}`, `{ctx}` (mgmt context name), `{time}` (HH:MM). Resolved at spawn time and substituted into the prompt string. Unknown tokens are left literal and produce a warning at validate time.

### Spawn modes

- **TUI (`tea.ExecProcess`)** — bubbletea suspends, the shell takes over the terminal directly (no PTY layer; we inherit stdio). Perfect terminal fidelity (colors, signals, raw-mode tools like `vim`/`k9s` all work). On exit, bubbletea resumes (or kapish exits in `--one-shot`).
- **Web UI (`creack/pty.Start`)** — backend allocates a PTY, bridges PTY ↔ WebSocket. xterm.js mounts in the `TerminalPanel` slot. Resize: xterm.js sends control frames; backend calls `pty.Setsize`. WebSocket close: backend sends SIGHUP to the shell process group, waits 30s, then `Kill()`.

### WebSocket protocol (`/api/v1/sessions/<id>/ws`)

Binary frames in both directions, 1-byte prefix:

| Prefix | Direction | Body | Meaning |
|---|---|---|---|
| `0x00` | client → server | bytes | stdin to PTY |
| `0x01` | client → server | JSON `{"cols":N,"rows":N}` | PTY resize |
| `0x02` | client → server | (empty) | ping |
| `0x00` | server → client | bytes | PTY stdout/stderr |
| `0x02` | server → client | (empty) | pong |

Heartbeat: client pings every 30s. Server closes idle sessions after 5 min.

### Cleanup

- Each spawn gets its own temp dir (mode `0700`) holding the kubeconfig and any init files.
- `defer os.RemoveAll(tempDir)` on the spawn function.
- Spawned shells run in their own process group; killing kapish sends SIGHUP to the group cleanly.
- `SIGINT`/`SIGTERM` on kapish triggers ordered shutdown: close watch, kill PTY children, clean temp dirs, exit.

## TUI design

bubbletea + bubbles (`list`, `textinput`, `spinner`, `help`) + lipgloss for styling.

### Cluster list screen

```
┌─ kapish ─ mgmt: prod-mgmt (12 clusters) ───────────────────┐
│ Filter: pro_                                                │
├─────────────────────────────────────────────────────────────┤
│   NAME            NS        PHASE         VERSION  PROVIDER │
│ ▸ prod-eu-1       prod      Provisioned   v1.30.2  aws      │
│   prod-us-east-1  prod      Provisioned   v1.30.2  aws      │
│   prod-us-west-1  prod      Failed        v1.29.4  aws  ⚠  │
├─────────────────────────────────────────────────────────────┤
│ ↑↓ navigate · / filter · ⏎ shell · r refresh · s settings  │
│ m switch mgmt · ? help · q quit                             │
└─────────────────────────────────────────────────────────────┘
```

### Model state

- `clusters []ClusterRow` (kept current via watch)
- `filter string` (live; fuzzy match across `name + namespace`)
- `cursor int` (selected row in filtered view; sticks to `namespace/name` across re-sorts)
- `phase Phase` (`loading | ready | error | spawning | settings`)
- `mgmtCtx string` (shown in title)

### Keymap

| Key | Action |
|---|---|
| ↑/↓ or k/j | Move cursor |
| g / G | Top / bottom |
| / | Enter filter mode (Esc cancels, Enter applies) |
| Enter | Spawn shell for selected cluster |
| r | Force refresh from mgmt cluster |
| s | Open settings screen |
| m | Open mgmt cluster picker |
| ? | Toggle help overlay |
| q or Ctrl+C | Quit |

### Phase coloring (lipgloss)

- `Provisioned` → green ✓
- `Provisioning` / `Pending` → yellow …
- `Failed` / `Deleting` → red ⚠

### Spawn flow

1. User presses Enter on a row.
2. If phase is `Failed` or `Deleting` → confirm modal ("Cluster is `Failed`. Spawn shell anyway? (y/N)").
3. Fetch kubeconfig Secret (with brief spinner).
4. Write temp kubeconfig + init file.
5. `tea.ExecProcess` to hand control to the shell.
6. On shell exit: cleanup, redraw list (or exit if `--one-shot`).

### Empty / error states

- No clusters → centered message: "No CAPI clusters found in `<ns or all>` on `<mgmt-ctx>`. Press `r` to refresh."
- Mgmt unreachable → centered error with underlying cause + "press `r` retry, `m` switch mgmt, `q` quit." If only one entry exists in `managementClusters.entries`, `m` still opens the picker (showing the single entry) — fixing the cluster requires `q` then `kapish config edit`.

### Settings screen

Press `s` → enter settings. Tabs (vertical list), matching the Web UI:

- Shell & prompt (shell command, prompt template)
- Env vars (KV editor)
- Aliases (KV editor)
- Working dir (text input)
- Theme (dark/light)

Form-based, navigated with arrow keys (←/→ between tabs, ↑/↓ within a form). Save (Ctrl+S or button) writes back to YAML. Inline validation errors block save. Esc cancels back to the cluster list.

### Mgmt cluster picker

Press `m` → modal overlay listing `managementClusters.entries[]` with the current entry marked. ↑/↓ to highlight, Enter to switch, Esc to cancel. Switching rebuilds the K8s client + watch and re-renders the cluster list. Live spawned shells keep their already-fetched kubeconfig and continue running.

## Web UI design

### Stack

React 18 + TypeScript + Tailwind + xterm.js. No external UI component library (per design handoff). Vite build; embedded into the Go binary via `//go:embed all:internal/web/frontend/dist/*`. `--dev` mode proxies `/` to `vite dev`.

### Visual design

Comes from the [Claude Design handoff](#design-handoff-reference) — KapishMark + KapishLockup logo, dark-default theme with light theme tokens, all Tailwind tokens specified, components for AppHeader / FilterInput / ClusterListRow / PhaseChip / TerminalPanel / EmptyState / ConfirmDialog / Toast / SettingsTabs / SettingsSectionForm / Field / KVList / ErrorBanner. Component code is the source of truth; this spec doesn't restate it.

### Routes / views

Single-route app with conditional render:

- **Main**: header + cluster list sidebar + terminal pane.
- **Settings**: header + tabbed full-width form.

### Main layout

```
┌──────────────────────────────────────────────────────────────────┐
│ [logo] kapish │ mgmt: prod-mgmt (12 clusters) │ ↻ refresh │ ⚙   │
├──────────────────────┬───────────────────────────────────────────┤
│ ⌕ Filter…            │  shell: prod-eu-1                  [×]    │
├──────────────────────┼───────────────────────────────────────────┤
│ ▸ prod-eu-1   Prov ✓ │  [prod-eu-1] $ kubectl get nodes           │
│   prod-us-e   Prov ✓ │  …                                         │
│   prod-us-w   Fail ⚠ │  [prod-eu-1] $ █                           │
└──────────────────────┴───────────────────────────────────────────┘
```

When no cluster is selected, the right pane shows `<SelectClusterEmpty />`. Switching clusters while a shell is open prompts via `<ConfirmDialog />` ("Disconnect current shell?"). Multiple shells via multiple browser tabs (no in-app tab management in v1).

### Settings page

Tabs (v1): Shell & prompt, Env vars, Aliases, Working dir, Theme. Tabs deferred to v2 (Hooks, Keybinds) are absent. Save POSTs to `/api/v1/config` with field-level validation; saved settings hot-reload per the table in the [Hot-reload semantics](#hot-reload-semantics) section. No "restart required" banner needed — none of the v1 settings tabs touch fields that require a restart.

### Mgmt cluster picker

The chip in `AppHeader` (mgmt label) is clickable. Dropdown shows all entries from `managementClusters.entries`, marks the current one. Picking a different entry rebuilds the K8s client + watch. Live PTY sessions keep their already-fetched kubeconfig and continue running.

### HTTP API (`/api/v1`)

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/clusters` | Snapshot of clusters from the watch cache |
| GET | `/api/v1/clusters/stream` | Server-Sent Events stream of add/update/delete events |
| GET | `/api/v1/mgmts` | List configured mgmt clusters + which is current |
| PUT | `/api/v1/mgmts/current` | Switch current mgmt cluster (body: `{"name":"..."}`) |
| POST | `/api/v1/sessions` | Create shell session (body: `{"namespace":"...","cluster":"..."}`); returns `{sessionID, wsURL, wsToken}` |
| GET | `/api/v1/sessions/<id>/ws` | WebSocket-PTY (requires one-time `?token=<wsToken>`) |
| GET | `/api/v1/config` | Read merged effective config |
| PUT | `/api/v1/config` | Write config (validated; comments preserved via yaml.v3 Node) |
| GET | `/api/v1/health` | Liveness |

### Security

- Bind to `127.0.0.1` by default. `--bind` to override; `0.0.0.0` prints a warning at startup.
- Session create returns a one-time WebSocket token; the WS endpoint requires the token. Stops cross-tab CSRF-style hijack.
- CORS: same-origin only.
- `X-Frame-Options: DENY` to block embedding.
- No auth backend, no telemetry. Single-user local tool.

### Build/embed flow

- `internal/web/frontend/` is a Vite project (TS + React + Tailwind + xterm.js).
- `make frontend` (or `pnpm build`) → `dist/`.
- Go embeds `dist/` via `//go:embed all:internal/web/frontend/dist/*`.
- `--dev` mode skips the embed and proxies to `vite dev`.

### Browser-tab parallelism

Each browser tab opens its own session via `POST /api/v1/sessions` and its own WebSocket. The cluster list is shared via a single in-process cache backed by the watch; tabs see consistent state via the SSE stream.

## Error handling

Every failure mode is surfaced visibly, never silently swallowed.

| Failure | Where | UX |
|---|---|---|
| Mgmt cluster unreachable (DNS/network) | Startup or refresh | TUI: full-screen error w/ underlying cause + `r` retry / `m` switch mgmt / `q` quit. Web UI: `MgmtUnreachableBanner` over cluster list with retry button. |
| Mgmt cluster auth expired (`Unauthorized`) | Anytime | Same banner with hint: "kubeconfig credentials may be expired; refresh SSO/AWS session." |
| CAPI CRDs not installed | Startup | Hard error with body: "Install Cluster API: `clusterctl init`". |
| Watch stream drops | Background | Auto-reconnect with exponential backoff (250ms → 5s cap). After 3 failed reconnects, surface a banner; keep last-known list visible. |
| Workload kubeconfig Secret missing | Cluster-select | `KubeconfigUnavailableBanner`; row shows warning glyph; spawn blocked. |
| RBAC denied | Startup or per-action | Specific message: "Need `get,list,watch` on `cluster.x-k8s.io/clusters`" + same for `secrets`. No silent degradation. |
| Shell binary not found (`exec.LookPath`) | Spawn | Pre-spawn check; "shell `<value>` not found in PATH" + link to settings. |
| PTY allocation fails | Web UI spawn | 500 to client; toast "Couldn't allocate terminal". Backend logs syscall error. |
| WebSocket drops mid-session | Web UI | Toast "Connection lost"; terminal pane shows reconnect affordance. Backend SIGHUPs the shell after 30s of disconnection. |
| Shell exits non-zero | Both | TUI: returns to list (status bar shows "[shell exited code N]"). Web UI: terminal shows "[disconnected: code N]" frozen until close. |
| Config file parse error | Startup | Don't crash — boot with built-in defaults and surface a banner. `kapish config edit` to fix. |
| Config validation error on save | Settings UI | Inline field errors; save blocked; existing config not touched. |
| Mgmt context switched live | Both | Existing PTY sessions keep running. New spawns use the new context. Banner: "Mgmt context switched. Existing shells unaffected." |

## Edge cases

- **Many clusters (>200)**: Web UI virtualizes the cluster list (windowing); `bubbles/list` is already efficient. Filter is debounced 100ms.
- **Long cluster names**: truncate with ellipsis; full name on hover (Web UI) or in a detail line below filter (TUI).
- **Duplicate names across namespaces**: row's secondary line shows namespace; selection keyed on `namespace/name`.
- **Cluster removed while shell is open**: shell keeps running with already-fetched kubeconfig. When user exits, the row is gone from the list.
- **Multiple browser tabs to one `kapish serve`**: each tab gets its own session. Cluster list shared via a single watch cache.
- **Cursor stickiness**: TUI cursor sticks to the same `namespace/name` across re-sorts.
- **Shell process group**: kapish spawns shells in their own process group so killing kapish sends SIGHUP cleanly without orphaning subprocesses.
- **Mgmt picker → unreachable mgmt**: validate before swapping; if unreachable, leave current mgmt active and show error toast.
- **Web UI bind to `0.0.0.0`**: explicit flag required, prints warning. No auth backend, so binding non-localhost is on the user.

## Observability

- **Structured logs** via `log/slog`, JSON output. Default level `info`; `--log-level debug` for troubleshooting.
- **Log file**: `$XDG_CACHE_HOME/kapish/kapish.log` (rotated at 10 MB, 3 generations). `--log-file -` writes to stderr.
- **No telemetry, no phone-home, no analytics.** Single-user local tool.
- **Web UI**: backend access log at `info`; per-session start/end + duration + cluster name at `info`; PTY stream content is **not** logged (privacy).

## Testing strategy

| Layer | What | How |
|---|---|---|
| Unit | Config load/merge/validate; init-script generation per shell; prompt-token rendering; kubeconfig temp-file lifecycle; phase coloring | `testing` + `testify`; table-driven tests |
| Integration | Listing clusters, fetching kubeconfig, watch reconnect | `kind` cluster + `clusterctl init` in CI; `testing` with longer timeouts |
| TUI E2E | Cluster-list rendering, filter, navigation, spawn → exit → return | `teatest` (charmbracelet's bubbletea harness) |
| Web E2E | Cluster list, spawn shell, type commands, exit, settings save | Playwright (TS) against `kapish serve --port N --no-open` in CI |
| Cross-shell | Init-script correctness for bash/zsh/fish | Each shell run with a generated init in CI matrix; assert `KUBECONFIG`, env vars, aliases, prompt prefix |
| Manual smoke | Mac + Linux; both TUI + Web; against a real management cluster | Pre-release checklist |

## v1 scope summary

In:
- TUI + Web UI from one binary
- Cluster list from a single mgmt cluster, refreshed via watch
- Mgmt cluster picker with multiple entries from config
- Shell spawn with kubeconfig + global env + global aliases + global prompt prefix + global cwd
- Bash, zsh, fish
- Settings UX in TUI + Web UI + `kapish config edit`
- Theme toggle (dark default + light)
- Error surfaces, observability, basic test coverage

Out (v2+):
- Pre/post shell hooks
- Customizable TUI keybindings
- nu, pwsh shells
- Per-cluster or per-namespace config overrides
- Label/pattern-based config rules
- `{git}` and `{region}` prompt tokens
- In-UI resource browsing

## Design handoff reference

The Web UI visual design — design tokens (Tailwind config + CSS variables), brand assets (KapishMark, KapishLockup), icon set, and all React component implementations (AppHeader, FilterInput, ClusterListRow, PhaseChip, TerminalPanel, EmptyState, ConfirmDialog, Toast, SettingsTabs, Field, KVList, SettingsSectionForm, ErrorBanner, ClusterListSkeleton) plus the App + SettingsView layouts — was produced via Claude Design and is the source of truth for visual implementation. The component code is to be dropped into `internal/web/frontend/src/` per the file map in the handoff.

The implementation should treat the handoff as authoritative for visual styling, layout, and component shape. Where the handoff implies features beyond v1 scope (Hooks tab, Keybinds tab, `nu`/`pwsh` shell options, per-cluster/per-namespace scope subtitle), those are omitted in v1 per the v1 scope summary above.

## Open questions

None — all reconciliation between the design and architectural decisions is captured under the **v1 scope summary** above. New design questions should land in a follow-up spec.
