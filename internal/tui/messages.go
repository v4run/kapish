package tui

import (
	"github.com/v4run/kapish/internal/capi"
	"github.com/v4run/kapish/internal/shell"
)

// clustersLoadedMsg is sent after a full LIST.
type clustersLoadedMsg struct{ clusters []capi.Cluster }

// clusterEventMsg carries one watch event.
type clusterEventMsg struct{ ev capi.Event }

// errMsg signals a fatal-for-now error (e.g. mgmt cluster unreachable).
type errMsg struct{ err error }

// shellExitedMsg is sent (via tea.ExecProcess callback) after a spawned shell exits.
type shellExitedMsg struct{ err error }

// refreshTickMsg fires on the periodic re-LIST timer.
type refreshTickMsg struct{}

// spawnReadyMsg carries a fully-prepared SpawnPlan. Update handles it by
// returning tea.ExecProcess so bubbletea can hand off the terminal.
type spawnReadyMsg struct{ plan *shell.SpawnPlan }

// spawnFailedMsg signals the shell could NOT be launched (kubeconfig fetch
// failed, shell binary missing, etc.) — as opposed to shellExitedMsg which
// carries the *shell's own* exit status after a successful launch.
type spawnFailedMsg struct{ err error }
