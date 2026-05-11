# kapish

A debugging tool for [Cluster API (CAPI)](https://cluster-api.sigs.k8s.io/). Lists workload clusters from a management cluster and drops you into a shell whose `KUBECONFIG` is pre-set, with configurable env vars, aliases, working directory, and prompt prefix. Runs as a TUI, or as a localhost web app with an in-browser terminal.

## Install

```sh
go install github.com/v4run/kapish/cmd/kapish@latest
```

Make sure `$(go env GOBIN)` (or `$(go env GOPATH)/bin` if `GOBIN` is unset) is in `PATH`. The web UI is embedded in the binary, so no Node toolchain is needed at install time.

Or build from source:

```sh
git clone https://github.com/v4run/kapish
cd kapish
make build
./bin/kapish version
```

## Usage

```sh
kapish                    # TUI: pick a cluster, drop into a shell
kapish serve              # web UI on localhost (opens your browser)
kapish version            # show version
kapish config validate    # show merged effective config
kapish config edit        # edit config in $EDITOR; validate on save
```

### TUI

`kapish` (no subcommand) launches the terminal UI. Keys: `↑↓`/`jk`/`gG` navigate, `/` filter, `⏎` spawn a shell for the selected cluster (confirm-on-Failed), `r` refresh, `m` switch management cluster, `s` settings, `q` quit. `--one-shot` exits after the first spawned shell instead of returning to the list.

### Web UI

`kapish serve` starts a localhost HTTP server (binds `127.0.0.1` by default) and opens your browser. The page shows the cluster list (live-updated via SSE); clicking a cluster opens an in-browser terminal (xterm.js over a WebSocket-PTY bridge). The settings page reads/writes config; the management-cluster chip switches between configured management clusters. `--port N` pins the port (default: a free one), `--bind` overrides the bind address (non-loopback prints a warning — there is no authentication), `--no-open` skips the browser launch.

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

Config lives at `~/.config/kapish/config.yaml` by default (resolution order: `--config` flag > `$KAPISH_CONFIG` > `$XDG_CONFIG_HOME/kapish/config.yaml`). On first run, `kapish config edit` writes a commented template there. The schema covers management clusters, shell command/cwd/env/aliases/prompt, UI theme/refresh, and web bind/port.

See [`docs/superpowers/specs/2026-05-09-kapish-design.md`](docs/superpowers/specs/2026-05-09-kapish-design.md) for the full design.

## Development

```sh
make build      # build ./bin/kapish (with version/commit ldflags)
make test       # go test ./...
make lint       # go vet ./...
make fmt        # go fmt ./...
make frontend   # rebuild the web UI: cd internal/web/frontend && npm install && npm run build
```

The web UI is a Vite + React + Tailwind + xterm.js app under `internal/web/frontend/`. The build output (`internal/web/frontend/dist/`) is committed and embedded into the Go binary via `//go:embed` — **after any change to `internal/web/frontend/src/`, run `make frontend` and commit the regenerated `dist/`.**

For frontend hot-reload during development:

```sh
# terminal 1: Go server (proxies / to the Vite dev server)
kapish serve --dev --kubeconfig <your kubeconfig>
# terminal 2: Vite dev server (proxies /api back to the Go server — set the port it printed)
VITE_KAPISH_API=http://127.0.0.1:<port-from-terminal-1> npm --prefix internal/web/frontend run dev
```

## Status

The TUI and web UI are both functional. Known v1 limitations: the fish-shell prompt prefix is overridden by a user's customized `fish_prompt`; settings editing in the TUI is read-only (theme toggle only) while the web UI has the full editor; CAPI integration targets the (now-deprecated) `v1beta1` API. See `docs/superpowers/plans/` for the implementation history.
