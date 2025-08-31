package serverclass

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
	"gopkg.in/ini.v1"
)

type Updater struct {
	path       string
	backup     bool
	dryRun     bool
	dryRunApps []string
	fs         afero.Fs
	// App class names are the [serverClass:...] stanzas; whitelist is "whitelist"
}
type Config struct {
	Path           string
	Backup         bool
	AppClass       map[string]string // app -> class name
	AppDestination map[string]string // app -> dest key
	DryRun         bool
	DryRunApps     []string
}

func NewUpdater(cfg Config) *Updater {
	return &Updater{path: cfg.Path, backup: cfg.Backup, dryRun: cfg.DryRun, dryRunApps: cfg.DryRunApps, fs: afero.NewOsFs()}
}

// SetFS allows overriding the filesystem (e.g., in tests) with an in-memory FS.
func (u *Updater) SetFS(fs afero.Fs) { u.fs = fs }

func (u *Updater) UpdateWhitelist(app string, serverClass string, patterns []string) error {
	if err := u.fs.MkdirAll(filepath.Dir(u.path), 0o755); err != nil {
		return err
	}
	var cfg *ini.File
	var err error
	if _, err = u.fs.Stat(u.path); err == nil {
		b, rerr := afero.ReadFile(u.fs, u.path)
		if rerr != nil {
			return rerr
		}
		cfg, err = ini.Load(b)
		if err != nil {
			return fmt.Errorf("load serverclass: %w", err)
		}
	} else if os.IsNotExist(err) {
		cfg = ini.Empty()
	} else {
		return err
	}

	secName := fmt.Sprintf("serverClass:%s", serverClass)
	sec, _ := cfg.GetSection(secName)
	if sec == nil {
		sec, _ = cfg.NewSection(secName)
	}

	sort.Strings(patterns)
	// Capture previous whitelist keys
	prev := collectWhitelist(sec)
	// Clear previous whitelist keys
	for _, k := range sec.Keys() {
		if strings.HasPrefix(k.Name(), "whitelist") {
			sec.DeleteKey(k.Name())
		}
	}
	// Write as whitelist.N entries (0-based)
	for i, p := range patterns {
		key := "whitelist." + strconv.Itoa(i)
		sec.Key(key).SetValue(p)
	}

	// Compute diff
	adds, removes := diffSets(prev, patterns)
	effectiveDryRun := u.dryRun || contains(u.dryRunApps, app)
	if effectiveDryRun {
		log.Printf("[dry-run][app:%s][class:%s] +%d, -%d (file: %s)", app, serverClass, len(adds), len(removes), u.path)
		return nil
	}
	if len(adds) == 0 && len(removes) == 0 {
		log.Printf("[noop][app:%s][class:%s] no whitelist changes", app, serverClass)
		return nil
	}
	// Perform timestamped backup (copy) if enabled and target exists
	if u.backup {
		if _, statErr := u.fs.Stat(u.path); statErr == nil {
			if _, err := u.makeTimestampBackup(); err != nil {
				return err
			}
		}
	}

	// Write to a temp file then atomically replace
	ts := time.Now().UTC().Format("20060102-150405")
	tmpPath := fmt.Sprintf("%s.tmp-%d-%s", u.path, os.Getpid(), ts)
	f, ferr := u.fs.Create(tmpPath)
	if ferr != nil {
		return ferr
	}
	_, werr := cfg.WriteTo(f)
	cerr := f.Close()
	if werr != nil {
		_ = u.fs.Remove(tmpPath)
		return werr
	}
	if cerr != nil {
		_ = u.fs.Remove(tmpPath)
		return cerr
	}
	if rerr := u.fs.Rename(tmpPath, u.path); rerr != nil {
		// rollback: original file still intact (we only wrote temp). Cleanup temp and bubble error
		_ = u.fs.Remove(tmpPath)
		return rerr
	}
	return nil
}

// makeTimestampBackup creates a copy of the current target file with a timestamped .bak suffix.
func (u *Updater) makeTimestampBackup() (string, error) {
	ts := time.Now().UTC().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s.bak", u.path, ts)
	src, err := u.fs.Open(u.path)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := u.fs.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = dst.Close() }()
	if _, err := io.Copy(dst, src); err != nil {
		_ = u.fs.Remove(backupPath)
		return "", err
	}
	return backupPath, nil
}

func collectWhitelist(sec *ini.Section) []string {
	var vals []string
	for _, k := range sec.Keys() {
		if strings.HasPrefix(k.Name(), "whitelist") {
			vals = append(vals, k.Value())
		}
	}
	sort.Strings(vals)
	return vals
}

func diffSets(oldList, newList []string) (adds, removes []string) {
	old := make(map[string]struct{}, len(oldList))
	for _, v := range oldList {
		old[v] = struct{}{}
	}
	newm := make(map[string]struct{}, len(newList))
	for _, v := range newList {
		newm[v] = struct{}{}
		if _, ok := old[v]; !ok {
			adds = append(adds, v)
		}
	}
	for v := range old {
		if _, ok := newm[v]; !ok {
			removes = append(removes, v)
		}
	}
	sort.Strings(adds)
	sort.Strings(removes)
	return
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
