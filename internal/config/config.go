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

type ServiceNowAuth struct {
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	BearerToken string `yaml:"bearerToken"`
}

type ServiceNowConfig struct {
	BaseURL            string         `yaml:"baseURL"`
	Table              string         `yaml:"table"`
	Query              string         `yaml:"query"`
	HostnameField      string         `yaml:"hostnameField"`
	LaneField          string         `yaml:"laneField"`
	PageSize           int            `yaml:"pageSize"`
	Timeout            Duration       `yaml:"timeout"`
	InsecureSkipVerify bool           `yaml:"insecureSkipVerify"`
	Auth               ServiceNowAuth `yaml:"auth"`
}

type CMDBConfig struct {
	Type       string           `yaml:"type"` // "dummy" or "servicenow"
	Dummy      DummyCMDBConfig  `yaml:"dummy"`
	ServiceNow ServiceNowConfig `yaml:"servicenow"`
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
	// Deprecated: use CMDB
	DummyCMDB   DummyCMDBConfig   `yaml:"dummyCMDB"`
	CMDB        CMDBConfig        `yaml:"cmdb"`
	Serverclass ServerclassConfig `yaml:"serverclass"`
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
	// Defaults for CMDB
	if cfg.CMDB.Type == "" {
		if len(cfg.DummyCMDB.Entries) > 0 {
			cfg.CMDB.Type = "dummy"
			cfg.CMDB.Dummy = cfg.DummyCMDB
		} else {
			cfg.CMDB.Type = "dummy"
		}
	}
	if cfg.CMDB.Type == "servicenow" {
		if cfg.CMDB.ServiceNow.PageSize == 0 {
			cfg.CMDB.ServiceNow.PageSize = 100
		}
		if cfg.CMDB.ServiceNow.Timeout.Duration == 0 {
			cfg.CMDB.ServiceNow.Timeout = Duration{Duration: 30 * time.Second}
		}
	}
	return &cfg, nil
}
