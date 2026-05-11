package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/capi"
	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/tui"
)

// runTUI is the default action when `kapish` is invoked with no subcommand.
func runTUI(cmd *cobra.Command, args []string) error {
	g, err := readGlobalFlags(cmd)
	if err != nil {
		return err
	}

	// Resolve + load config.
	cfgPath, err := kconfig.ResolvePath(kconfig.PathSources{
		Flag:          g.ConfigPath,
		EnvVar:        os.Getenv("KAPISH_CONFIG"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		Home:          os.Getenv("HOME"),
	})
	if err != nil {
		return err
	}
	app, err := kconfig.LoadFromFile(cfgPath)
	if err != nil {
		return err
	}
	app = kconfig.ApplyOverrides(app, kconfig.FlagOverrides{
		Kubeconfig: g.Kubeconfig,
		Context:    g.Context,
		OneShot:    boolPtrIfSet(cmd, "one-shot", g.OneShot),
	})
	if err := kconfig.Validate(app); err != nil {
		return err
	}

	// Best-effort sweep of stale temp dirs from prior runs.
	_, _ = sweepStaleTempDirs(os.TempDir(), 24*time.Hour)

	// Build the capi client from the current mgmt entry (or flags).
	mgmtKubeconfig := g.Kubeconfig
	mgmtContextName := g.Context
	if idx := indexOfCurrentEntry(app); idx >= 0 {
		e := app.ManagementClusters.Entries[idx]
		if mgmtKubeconfig == "" {
			mgmtKubeconfig = e.Kubeconfig
		}
		if mgmtContextName == "" {
			mgmtContextName = e.Context
		}
	}
	client, err := capi.NewClient(capi.Options{
		Kubeconfig: mgmtKubeconfig,
		Context:    mgmtContextName,
		Namespace:  currentNamespace(app),
	})
	if err != nil {
		return fmt.Errorf("connect to management cluster: %w", err)
	}

	model := tui.New(tui.Config{
		CapiClient:  client,
		AppConfig:   app,
		MgmtContext: client.Context(),
		OneShot:     app.UI.OneShot,
		ConfigPath:  cfgPath,
	})
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := finalModel.(tui.Model); ok {
		if fe := m.FatalErr(); fe != nil {
			return fe
		}
	}
	return nil
}

func indexOfCurrentEntry(c kconfig.Config) int {
	cur := c.ManagementClusters.Current
	if cur == "" {
		if len(c.ManagementClusters.Entries) == 1 {
			return 0
		}
		return -1
	}
	for i, e := range c.ManagementClusters.Entries {
		if e.Name == cur {
			return i
		}
	}
	return -1
}

func currentNamespace(c kconfig.Config) string {
	if idx := indexOfCurrentEntry(c); idx >= 0 {
		return c.ManagementClusters.Entries[idx].Namespace
	}
	return ""
}
