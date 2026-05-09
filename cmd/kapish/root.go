package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kapish",
		Short: "kapish — pick a CAPI cluster and drop into a shell",
		Long: `kapish lists Cluster API workload clusters from a management
cluster and lets you drop into a shell with KUBECONFIG, aliases,
env vars, and a prompt scoped to the chosen cluster.

Run "kapish" (no args) for the TUI, or "kapish serve" for the web UI.`,
		SilenceUsage: true,
	}
	return root
}
