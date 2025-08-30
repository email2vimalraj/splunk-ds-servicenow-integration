package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/splunk-ds-camr/internal/cmdb"
	sn "github.com/example/splunk-ds-camr/internal/cmdb/servicenow"
	"github.com/example/splunk-ds-camr/internal/config"
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
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var cmdbClient cmdb.Client
	switch cfg.CMDB.Type {
	case "servicenow":
		cmdbClient = sn.New(cfg.CMDB.ServiceNow)
	default:
		cmdbClient = cmdb.NewDummy(cfg.CMDB.Dummy)
	}
	updater := serverclass.NewUpdater(serverclass.Config{
		Path:           cfg.Serverclass.Path,
		Backup:         cfg.Serverclass.Backup,
		AppClass:       cfg.Serverclass.AppClass,
		AppDestination: cfg.Serverclass.AppDestination,
	})

	// One-shot mode for testing/automation
	if once := os.Getenv("CAMR_ONCE"); once == "1" || once == "true" {
		if err := runOnce(ctx, cmdbClient, updater, cfg); err != nil {
			log.Fatalf("run once: %v", err)
		}
		log.Println("completed single run")
		return
	}

	ticker := time.NewTicker(cfg.RefreshInterval.Duration)
	defer ticker.Stop()

	log.Printf("starting with refresh interval %s", cfg.RefreshInterval.Duration)
	for {
		if err := runOnce(ctx, cmdbClient, updater, cfg); err != nil {
			log.Printf("error: %v", err)
		}

		select {
		case <-ctx.Done():
			log.Println("shutting down")
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
		patternsByDest[dest] = patterns.GenerateWildcards(hosts)
	}

	// update serverclass files for each destination mapping
	for app, class := range cfg.Serverclass.AppClass {
		dest := cfg.Serverclass.AppDestination[app]
		if dest == "" {
			continue
		}
		patterns := patternsByDest[dest]
		if err := u.UpdateWhitelist(class, patterns); err != nil {
			return err
		}
	}
	return nil
}
