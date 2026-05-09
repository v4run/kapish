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
