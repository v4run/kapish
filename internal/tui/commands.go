package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/kapish/internal/capi"
	"github.com/v4run/kapish/internal/shell"
)

// loadCmd performs a full LIST and emits clustersLoadedMsg (or errMsg).
func (m Model) loadCmd() tea.Cmd {
	client := m.cfg.CapiClient
	if client == nil {
		return func() tea.Msg { return clustersLoadedMsg{} }
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cs, err := client.ListClusters(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return clustersLoadedMsg{clusters: cs}
	}
}

// refreshTickCmd returns a cmd that fires refreshTickMsg after the configured
// refresh interval (default 30s).
func (m Model) refreshTickCmd() tea.Cmd {
	d := time.Duration(m.cfg.AppConfig.UI.RefreshIntervalSec) * time.Second
	if d <= 0 {
		d = 30 * time.Second
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// spawnCmd fetches the kubeconfig and prepares the shell plan, emitting
// spawnReadyMsg on success or shellExitedMsg on any prep error.
//
// Note: tea.ExecProcess must be returned as a tea.Cmd — NOT called immediately.
// This two-step pattern (spawnCmd → spawnReadyMsg → tea.ExecProcess in Update)
// is required so bubbletea can handle the terminal handoff specially.
func (m Model) spawnCmd(c capi.Cluster) tea.Cmd {
	client := m.cfg.CapiClient
	app := m.cfg.AppConfig
	mgmtCtx := m.mgmtContext
	return func() tea.Msg {
		if client == nil {
			return shellExitedMsg{err: fmt.Errorf("tui: no capi client")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		kc, err := client.FetchKubeconfig(ctx, c.Namespace, c.Name)
		cancel()
		if err != nil {
			return shellExitedMsg{err: err}
		}
		opts := shell.Options{
			PathToShell:    app.Shell.Command,
			Cwd:            app.Shell.Cwd,
			Env:            app.Shell.Env,
			Aliases:        app.Shell.Aliases,
			PromptTemplate: app.Shell.Prompt,
			PromptTokens: shell.PromptTokens{
				Cluster:   c.Name,
				Namespace: c.Namespace,
				Provider:  c.Provider,
				Ctx:       mgmtCtx,
				Now:       time.Now(),
			},
		}
		plan, err := shell.PrepareSpawn(opts, kc)
		if err != nil {
			return shellExitedMsg{err: err}
		}
		return spawnReadyMsg{plan: plan}
	}
}
