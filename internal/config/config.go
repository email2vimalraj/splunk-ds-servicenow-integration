package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration struct{ time.Duration }

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

type DummyCMDBEntry struct {
	Hostname            string `yaml:"hostname"`
	BusinessServiceLane string `yaml:"businessServiceLane"`
}

type DummyCMDBConfig struct {
	Entries []DummyCMDBEntry `yaml:"entries"`
}

type ServerclassConfig struct {
	Path           string            `yaml:"path"`
	Backup         bool              `yaml:"backup"`
	AppClass       map[string]string `yaml:"appClass"`
	AppDestination map[string]string `yaml:"appDestination"`
}

type Config struct {
	RefreshInterval Duration            `yaml:"refreshInterval"`
	Destinations    map[string][]string `yaml:"destinations"`
	DummyCMDB       DummyCMDBConfig     `yaml:"dummyCMDB"`
	Serverclass     ServerclassConfig   `yaml:"serverclass"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if cfg.RefreshInterval.Duration == 0 {
		cfg.RefreshInterval = Duration{Duration: 5 * time.Minute}
	}
	return &cfg, nil
}
