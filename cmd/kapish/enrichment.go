package main

import (
	"log/slog"
	"time"

	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/enrich"
)

// buildEnrichmentManager maps enrichment config to an enrich.Manager. It is
// defensive: a provider or column it can't build is logged and skipped so a bad
// block never breaks the cluster list. Returns nil when nothing usable remains.
func buildEnrichmentManager(cfg kconfig.EnrichmentConfig) *enrich.Manager {
	valid := make(map[string]bool)
	var specs []enrich.ProviderSpec
	for _, p := range cfg.Providers {
		if p.Type != "shell" || len(p.Command) == 0 || p.Key == "" || p.RefreshIntervalSec <= 0 || p.TimeoutSec <= 0 {
			slog.Warn("enrichment: skipping invalid provider", "name", p.Name, "type", p.Type)
			continue
		}
		specs = append(specs, enrich.ProviderSpec{
			Provider: enrich.NewShellProvider(p.Name, p.Command, p.Key),
			Interval: time.Duration(p.RefreshIntervalSec) * time.Second,
			Timeout:  time.Duration(p.TimeoutSec) * time.Second,
		})
		valid[p.Name] = true
	}

	var cols []enrich.Column
	for _, c := range cfg.Columns {
		if !valid[c.Provider] || c.Field == "" || (c.JoinOn != "name" && c.JoinOn != "namespace") {
			slog.Warn("enrichment: skipping invalid column", "header", c.Header, "provider", c.Provider)
			continue
		}
		cols = append(cols, enrich.Column{
			Header:   c.Header,
			Provider: c.Provider,
			Field:    c.Field,
			JoinOn:   c.JoinOn,
			Width:    c.Width,
		})
	}

	if len(specs) == 0 || len(cols) == 0 {
		return nil
	}
	return enrich.NewManager(specs, cols)
}
