package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/capi"
	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/web"
)

func newServeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "serve",
		Short: "Run the kapish web UI on localhost",
		RunE:  runServe,
	}
	c.Flags().Int("port", 0, "Port to bind (0 = pick a free port)")
	c.Flags().String("bind", "127.0.0.1", "Address to bind (non-loopback prints a warning)")
	c.Flags().Bool("no-open", false, "Don't open the browser automatically")
	c.Flags().Bool("dev", false, "Dev mode (reserved for Vite proxy; currently a no-op)")
	return c
}

func runServe(cmd *cobra.Command, args []string) error {
	g, err := readGlobalFlags(cmd)
	if err != nil {
		return err
	}
	port, _ := cmd.Flags().GetInt("port")
	bind, _ := cmd.Flags().GetString("bind")
	noOpen, _ := cmd.Flags().GetBool("no-open")

	cfgPath, err := kconfig.ResolvePath(kconfig.PathSources{
		Flag: g.ConfigPath, EnvVar: os.Getenv("KAPISH_CONFIG"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"), Home: os.Getenv("HOME"),
	})
	if err != nil {
		return err
	}
	app, err := kconfig.LoadFromFile(cfgPath)
	if err != nil {
		return err
	}
	app = kconfig.ApplyOverrides(app, kconfig.FlagOverrides{
		Kubeconfig: g.Kubeconfig, Context: g.Context, OneShot: boolPtrIfSet(cmd, "one-shot", g.OneShot),
	})
	if err := kconfig.Validate(app); err != nil {
		return err
	}

	// Best-effort sweep of stale temp dirs from prior runs.
	_, _ = sweepStaleTempDirs(os.TempDir(), 24*time.Hour)

	if bind != "127.0.0.1" && bind != "localhost" && bind != "::1" {
		fmt.Fprintf(os.Stderr, "warning: binding to %s exposes kapish with no authentication\n", bind)
	}

	mgmtKubeconfig, mgmtContext := g.Kubeconfig, g.Context
	mgmtNamespace := ""
	if idx := indexOfCurrentEntry(app); idx >= 0 {
		e := app.ManagementClusters.Entries[idx]
		if mgmtKubeconfig == "" {
			mgmtKubeconfig = e.Kubeconfig
		}
		if mgmtContext == "" {
			mgmtContext = e.Context
		}
		mgmtNamespace = e.Namespace
	}
	client, err := capi.NewClient(capi.Options{Kubeconfig: mgmtKubeconfig, Context: mgmtContext, Namespace: mgmtNamespace})
	if err != nil {
		return fmt.Errorf("connect to management cluster: %w", err)
	}

	srv, err := web.New(web.Options{
		CapiClient: client, AppConfig: app, MgmtContext: client.Context(),
		ConfigPath: cfgPath, BindAddr: bind, Port: port,
	})
	if err != nil {
		return err
	}
	addr, err := srv.Listen()
	if err != nil {
		return err
	}
	url := "http://" + addr + "/"
	fmt.Println("kapish web UI:", url)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Seed + watch + periodic re-list.
	if cs, lerr := client.ListClusters(rootCtx); lerr == nil {
		srv.ReplaceAll(cs)
	}
	go func() {
		for rootCtx.Err() == nil {
			evs, werr := client.WatchClusters(rootCtx)
			if werr != nil {
				time.Sleep(time.Second)
				continue
			}
			for ev := range evs {
				srv.ApplyEvent(ev)
			}
			// channel closed; loop reconnects unless ctx done.
			time.Sleep(time.Second)
		}
	}()
	go func() {
		d := time.Duration(app.UI.RefreshIntervalSec) * time.Second
		if d <= 0 {
			d = 30 * time.Second
		}
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-t.C:
				if cs, lerr := client.ListClusters(rootCtx); lerr == nil {
					srv.ReplaceAll(cs)
				}
			}
		}
	}()

	if !noOpen && app.Web.OpenBrowser {
		_ = openBrowser(url)
	}

	srvErr := make(chan error, 1)
	go func() { srvErr <- srv.Serve() }()

	select {
	case <-rootCtx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	case err := <-srvErr:
		return err
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
