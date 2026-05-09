package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kapish version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(version.Long())
			return nil
		},
	}
}
