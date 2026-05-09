package main

import "github.com/spf13/cobra"

// GlobalFlags is the typed read-out of the cobra root's persistent flags.
// All subcommands extract their settings via this struct so flag handling
// stays in one place.
type GlobalFlags struct {
	ConfigPath string
	Kubeconfig string
	Context    string
	LogLevel   string
	LogFile    string
	OneShot    bool
}

func registerGlobalFlags(cmd *cobra.Command) {
	pf := cmd.PersistentFlags()
	pf.String("config", "", "Path to kapish config (overrides $KAPISH_CONFIG and XDG defaults)")
	pf.String("kubeconfig", "", "Path to kubeconfig for the management cluster (overrides config)")
	pf.String("context", "", "kubeconfig context for the management cluster (overrides config)")
	pf.String("log-level", "info", "Log level: debug | info | warn | error")
	pf.String("log-file", "", "Log file path. Use '-' for stderr. Empty = $XDG_CACHE_HOME/kapish/kapish.log.")
	pf.Bool("one-shot", false, "TUI exits after first spawned shell exits, instead of returning to list")
}

func readGlobalFlags(cmd *cobra.Command) (GlobalFlags, error) {
	pf := cmd.Flags()
	g := GlobalFlags{}
	var err error
	if g.ConfigPath, err = pf.GetString("config"); err != nil {
		return g, err
	}
	if g.Kubeconfig, err = pf.GetString("kubeconfig"); err != nil {
		return g, err
	}
	if g.Context, err = pf.GetString("context"); err != nil {
		return g, err
	}
	if g.LogLevel, err = pf.GetString("log-level"); err != nil {
		return g, err
	}
	if g.LogFile, err = pf.GetString("log-file"); err != nil {
		return g, err
	}
	if g.OneShot, err = pf.GetBool("one-shot"); err != nil {
		return g, err
	}
	return g, nil
}
