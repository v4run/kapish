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
