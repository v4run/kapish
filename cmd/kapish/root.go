package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/kapishlog"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runTUI,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			g, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			path := g.LogFile
			if path == "" {
				cache := os.Getenv("XDG_CACHE_HOME")
				if cache == "" {
					cache = filepath.Join(os.Getenv("HOME"), ".cache")
				}
				path = filepath.Join(cache, "kapish", "kapish.log")
			}
			logger, err := kapishlog.New(kapishlog.Options{Level: g.LogLevel, FilePath: path})
			if err != nil {
				return err
			}
			slog.SetDefault(logger)
			return nil
		},
	}
	registerGlobalFlags(root)
	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd())
	return root
}
