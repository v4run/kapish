package tui

import "github.com/v4run/kapish/internal/capi"

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
