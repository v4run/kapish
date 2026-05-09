package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/v4run/kapish/internal/config"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Inspect or edit kapish configuration",
	}
	c.AddCommand(newConfigValidateCmd())
	c.AddCommand(newConfigEditCmd())
	return c
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate config and print the merged effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			path, err := config.ResolvePath(config.PathSources{
				Flag:          g.ConfigPath,
				EnvVar:        os.Getenv("KAPISH_CONFIG"),
				XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
				Home:          os.Getenv("HOME"),
			})
			if err != nil {
				return err
			}

			cfg, err := config.LoadFromFile(path)
			if err != nil {
				return err
			}

			cfg = config.ApplyOverrides(cfg, config.FlagOverrides{
				Kubeconfig: g.Kubeconfig,
				Context:    g.Context,
				OneShot:    boolPtrIfSet(cmd, "one-shot", g.OneShot),
			})

			if err := config.Validate(cfg); err != nil {
				return err
			}

			out, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "# Effective kapish config (path:", path+")")
			fmt.Fprint(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the kapish config in $EDITOR; validate after save",
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := readGlobalFlags(cmd)
			if err != nil {
				return err
			}
			path, err := config.ResolvePath(config.PathSources{
				Flag:          g.ConfigPath,
				EnvVar:        os.Getenv("KAPISH_CONFIG"),
				XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
				Home:          os.Getenv("HOME"),
			})
			if err != nil {
				return err
			}

			// Make sure the file exists with a useful template if needed.
			if _, err := config.EnsureFirstRunTemplate(path); err != nil {
				return err
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				return errors.New("$EDITOR (or $VISUAL) is not set")
			}

			ed := exec.Command(editor, path)
			ed.Stdin = os.Stdin
			ed.Stdout = os.Stdout
			ed.Stderr = os.Stderr
			if err := ed.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			cfg, err := config.LoadFromFile(path)
			if err != nil {
				return err
			}
			if err := config.Validate(cfg); err != nil {
				return fmt.Errorf("edited config is invalid:\n%w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Saved:", path)
			return nil
		},
	}
}

// boolPtrIfSet returns &val when the named flag was explicitly provided on
// the command line, nil otherwise.
func boolPtrIfSet(cmd *cobra.Command, name string, val bool) *bool {
	f := cmd.Flag(name)
	if f == nil || !f.Changed {
		return nil
	}
	return &val
}
