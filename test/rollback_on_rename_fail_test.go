package test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/example/splunk-ds-camr/internal/cmdb"
	"github.com/example/splunk-ds-camr/internal/config"
	"github.com/example/splunk-ds-camr/internal/serverclass"
)

type failRenameFs struct{ afero.Fs }

func (f failRenameFs) Rename(oldname, newname string) error {
	return errors.New("simulated rename failure")
}

func TestRollbackOnRenameFailure(t *testing.T) {
	// Prepare config and initial file
	cfg := &config.Config{
		Destinations: map[string][]string{
			"dest1": {"lane1"},
		},
		CMDB: config.CMDBConfig{Type: "dummy", Dummy: config.DummyCMDBConfig{Entries: []config.DummyCMDBEntry{
			{Hostname: "abc001", BusinessServiceLane: "lane1"},
			{Hostname: "abc002", BusinessServiceLane: "lane1"},
		}}},
		Serverclass: config.ServerclassConfig{
			Path:   "/serverclass.conf",
			Backup: true,
			AppClass: map[string]string{
				"AA-DESTINATION-dest1": "AA-DESTINATION-dest1-class",
			},
			AppDestination: map[string]string{
				"AA-DESTINATION-dest1": "dest1",
			},
		},
	}
	mem := afero.NewMemMapFs()
	// seed original file
	if err := afero.WriteFile(mem, cfg.Serverclass.Path, []byte("[serverClass:AA-DESTINATION-dest1-class]\nwhitelist.0 = oldvalue\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := cmdb.NewDummy(cfg.CMDB.Dummy)
	u := serverclass.NewUpdater(serverclass.Config{
		Path:           cfg.Serverclass.Path,
		Backup:         cfg.Serverclass.Backup,
		AppClass:       cfg.Serverclass.AppClass,
		AppDestination: cfg.Serverclass.AppDestination,
	})
	// wrap fs with failing rename
	u.SetFS(failRenameFs{Fs: mem})

	entries, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Build lane->dest
	destByLane := map[string]string{"lane1": "dest1"}
	hostsByDest := map[string][]string{"dest1": {}}
	for _, e := range entries {
		hostsByDest[destByLane[e.BusinessServiceLane]] = append(hostsByDest[destByLane[e.BusinessServiceLane]], e.Hostname)
	}
	for app, class := range cfg.Serverclass.AppClass {
		pats := []string{"abc*"}
		err := u.UpdateWhitelist(app, class, pats)
		if err == nil {
			t.Fatalf("expected error due to simulated rename failure, got nil")
		}
	}
	// original file should remain unchanged
	b, err := afero.ReadFile(mem, cfg.Serverclass.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "whitelist.0 = oldvalue") {
		t.Fatalf("original file was modified unexpectedly:\n%s", content)
	}
	// backup should exist
	dirEntries, err := afero.ReadDir(mem, "/")
	if err != nil {
		t.Fatal(err)
	}
	foundBak := false
	for _, de := range dirEntries {
		if strings.HasPrefix(de.Name(), "serverclass.conf.") && strings.HasSuffix(de.Name(), ".bak") {
			foundBak = true
			break
		}
	}
	if !foundBak {
		t.Fatalf("expected a timestamped .bak file to be created")
	}
	// temp file should be cleaned up
	for _, de := range dirEntries {
		if strings.HasPrefix(de.Name(), "serverclass.conf.tmp-") {
			t.Fatalf("temp file was not cleaned up: %s", de.Name())
		}
	}
}
