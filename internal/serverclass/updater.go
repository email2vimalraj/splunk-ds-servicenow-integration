package serverclass

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type Updater struct {
	path   string
	backup bool
	// App class names are the [serverClass:...] stanzas; whitelist is "whitelist"
}

type Config struct {
	Path           string
	Backup         bool
	AppClass       map[string]string // app -> class name
	AppDestination map[string]string // app -> dest key
}

func NewUpdater(cfg Config) *Updater {
	return &Updater{path: cfg.Path, backup: cfg.Backup}
}

func (u *Updater) UpdateWhitelist(serverClass string, patterns []string) error {
	if err := os.MkdirAll(filepath.Dir(u.path), 0o755); err != nil {
		return err
	}
	var cfg *ini.File
	var err error
	if _, err = os.Stat(u.path); err == nil {
		cfg, err = ini.Load(u.path)
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

	if u.backup {
		_ = os.Rename(u.path, u.path+".bak")
	}
	return cfg.SaveTo(u.path)
}
