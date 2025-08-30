package test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/example/splunk-ds-camr/internal/cmdb"
	"github.com/example/splunk-ds-camr/internal/config"
	"github.com/example/splunk-ds-camr/internal/patterns"
	"github.com/example/splunk-ds-camr/internal/serverclass"
)

func TestIntegration_DummyMemFS(t *testing.T) {
	// config: dest1 has lane1, dest2 has lane3
	cfg := &config.Config{
		RefreshInterval: config.Duration{},
		DryRun:          false,
		Destinations: map[string][]string{
			"dest1": {"lane1"},
			"dest2": {"lane3"},
		},
		CMDB: config.CMDBConfig{Type: "dummy", Dummy: config.DummyCMDBConfig{Entries: []config.DummyCMDBEntry{
			{Hostname: "abc001", BusinessServiceLane: "lane1"},
			{Hostname: "abc002", BusinessServiceLane: "lane1"},
			{Hostname: "xyz101", BusinessServiceLane: "lane3"},
			{Hostname: "xyz102", BusinessServiceLane: "lane3"},
		}}},
		Serverclass: config.ServerclassConfig{
			Path:   "/serverclass.conf",
			Backup: false,
			AppClass: map[string]string{
				"AA-DESTINATION-dest1": "AA-DESTINATION-dest1-class",
				"AA-DESTINATION-dest2": "AA-DESTINATION-dest2-class",
			},
			AppDestination: map[string]string{
				"AA-DESTINATION-dest1": "dest1",
				"AA-DESTINATION-dest2": "dest2",
			},
		},
	}

	fs := afero.NewMemMapFs()
	c := cmdb.NewDummy(cfg.CMDB.Dummy)
	u := serverclass.NewUpdater(serverclass.Config{
		Path:           cfg.Serverclass.Path,
		Backup:         cfg.Serverclass.Backup,
		AppClass:       cfg.Serverclass.AppClass,
		AppDestination: cfg.Serverclass.AppDestination,
	})
	u.SetFS(fs)

	entries, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Build lane->dest
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
			continue
		}
		hostsByDest[dest] = append(hostsByDest[dest], e.Hostname)
	}
	patternsByDest := map[string][]string{}
	for dest, hosts := range hostsByDest {
		patternsByDest[dest] = patterns.GenerateWildcards(hosts)
	}
	for app, class := range cfg.Serverclass.AppClass {
		dest := cfg.Serverclass.AppDestination[app]
		pats := patternsByDest[dest]
		if err := u.UpdateWhitelist(app, class, pats); err != nil {
			t.Fatal(err)
		}
	}
	// Read the file from memfs and assert content
	b, err := afero.ReadFile(fs, cfg.Serverclass.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !containsAll(content, []string{
		"[serverClass:AA-DESTINATION-dest1-class]",
		"whitelist.0 = abc*",
		"[serverClass:AA-DESTINATION-dest2-class]",
		"whitelist.0 = xyz*",
	}) {
		t.Fatalf("unexpected serverclass.conf contents:\n%s", content)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
