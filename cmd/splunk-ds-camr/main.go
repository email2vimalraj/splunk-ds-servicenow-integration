package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/splunk-ds-camr/internal/cmdb"
	sn "github.com/example/splunk-ds-camr/internal/cmdb/servicenow"
	"github.com/example/splunk-ds-camr/internal/config"
	"github.com/example/splunk-ds-camr/internal/logging"
	"github.com/example/splunk-ds-camr/internal/patterns"
	"github.com/example/splunk-ds-camr/internal/serverclass"
)

func main() {
	cfgPath := "config.yaml"
	if v := os.Getenv("CAMR_CONFIG"); v != "" {
		cfgPath = v
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		// can't init logger yet; print to stderr and exit
		panic(err)
	}

	// Initialize structured logging
	cleanup := logging.Init(cfg.Logging)
	defer cleanup()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var cmdbClient cmdb.Client
	switch cfg.CMDB.Type {
	case "servicenow":
		cmdbClient = sn.New(cfg.CMDB.ServiceNow)
	default:
		cmdbClient = cmdb.NewDummy(cfg.CMDB.Dummy)
	}
	// Environment override for dry-run
	if v := os.Getenv("CAMR_DRY_RUN"); v == "1" || v == "true" {
		cfg.DryRun = true
	}

	updater := serverclass.NewUpdater(serverclass.Config{
		Path:           cfg.Serverclass.Path,
		Backup:         cfg.Serverclass.Backup,
		AppClass:       cfg.Serverclass.AppClass,
		AppDestination: cfg.Serverclass.AppDestination,
		DryRun:         cfg.DryRun,
		DryRunApps:     cfg.Serverclass.DryRunApps,
	})

	// One-shot mode for testing/automation
	if once := os.Getenv("CAMR_ONCE"); once == "1" || once == "true" {
		if err := runOnce(ctx, cmdbClient, updater, cfg); err != nil {
			slog.Error("run once failed", "err", err)
			os.Exit(1)
		}
		slog.Info("completed single run")
		return
	}

	ticker := time.NewTicker(cfg.RefreshInterval.Duration)
	defer ticker.Stop()

	slog.Info("starting", "refreshInterval", cfg.RefreshInterval.Duration.String())
	for {
		if err := runOnce(ctx, cmdbClient, updater, cfg); err != nil {
			slog.Error("run error", "err", err)
		}

		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return
		case <-ticker.C:
		}
	}
}

func runOnce(ctx context.Context, c cmdb.Client, u *serverclass.Updater, cfg *config.Config) error {
	entries, err := c.Fetch(ctx)
	if err != nil {
		return err
	}

	// Build lane -> destination map from destination -> lanes config
	destByLane := map[string]string{}
	for dest, lanes := range cfg.Destinations {
		for _, lane := range lanes {
			destByLane[lane] = dest
		}
	}
	hostsByDest := map[string][]string{}
	for _, e := range entries {
		dest := destByLane[e.BusinessServiceLane]
		if dest == "" {
			continue // skip unknown lanes
		}
		hostsByDest[dest] = append(hostsByDest[dest], e.Hostname)
	}

	// compress hosts into wildcard patterns per destination
	patternsByDest := map[string][]string{}
	for dest, hosts := range hostsByDest {
		opts := patterns.Options{
			Mode:                  cfg.Wildcard.Mode,
			MinGroupSize:          cfg.Wildcard.MinGroupSize,
			RequireMinFixedPrefix: cfg.Wildcard.RequireMinFixedPrefix,
		}
		patternsByDest[dest] = patterns.GenerateWildcardsWithOptions(hosts, opts)
	}

	// update serverclass files for each destination mapping
	for app, class := range cfg.Serverclass.AppClass {
		dest := cfg.Serverclass.AppDestination[app]
		if dest == "" {
			continue
		}
		patterns := patternsByDest[dest]
		if err := u.UpdateWhitelist(app, class, patterns); err != nil {
			return err
		}
	}
	return nil
}
