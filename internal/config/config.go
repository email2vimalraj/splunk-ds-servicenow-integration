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
	DryRunApps     []string          `yaml:"dryRunApps"`
}

type WildcardConfig struct {
	Mode                  string `yaml:"mode"`                  // "trailingOnly" | "internalNumeric"
	MinGroupSize          int    `yaml:"minGroupSize"`          // default 2
	RequireMinFixedPrefix int    `yaml:"requireMinFixedPrefix"` // default 0
}

type LoggingConfig struct {
	// JSON structured logging to file with rotation
	Level      string `yaml:"level"`      // debug|info|warn|error (default: info)
	File       string `yaml:"file"`       // path to log file (default: ./camr.log)
	MaxSizeMB  int    `yaml:"maxSizeMB"`  // rotate after size in MB (default: 100)
	MaxBackups int    `yaml:"maxBackups"` // number of rotated files to keep (default: 7; 0 to keep unlimited)
	MaxAgeDays int    `yaml:"maxAgeDays"` // days to retain old logs (default: 30; 0 to keep forever)
	Compress   *bool  `yaml:"compress"`   // gzip old logs (default: true)
	Stdout     *bool  `yaml:"stdout"`     // also log to stdout (default: true)
}

type Config struct {
	RefreshInterval Duration            `yaml:"refreshInterval"`
	DryRun          bool                `yaml:"dryRun"`
	Destinations    map[string][]string `yaml:"destinations"`
	// Deprecated: use CMDB
	DummyCMDB   DummyCMDBConfig   `yaml:"dummyCMDB"`
	CMDB        CMDBConfig        `yaml:"cmdb"`
	Serverclass ServerclassConfig `yaml:"serverclass"`
	Wildcard    WildcardConfig    `yaml:"wildcard"`
	Logging     LoggingConfig     `yaml:"logging"`
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
	// Wildcard defaults
	if cfg.Wildcard.Mode == "" {
		cfg.Wildcard.Mode = "trailingOnly"
	}
	if cfg.Wildcard.MinGroupSize <= 0 {
		cfg.Wildcard.MinGroupSize = 2
	}
	if cfg.Wildcard.RequireMinFixedPrefix < 0 {
		cfg.Wildcard.RequireMinFixedPrefix = 0
	}
	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.File == "" {
		cfg.Logging.File = "./camr.log"
	}
	if cfg.Logging.MaxSizeMB <= 0 {
		cfg.Logging.MaxSizeMB = 100
	}
	if cfg.Logging.MaxBackups < 0 {
		cfg.Logging.MaxBackups = 7
	} else if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 7
	}
	if cfg.Logging.MaxAgeDays < 0 {
		cfg.Logging.MaxAgeDays = 30
	} else if cfg.Logging.MaxAgeDays == 0 {
		cfg.Logging.MaxAgeDays = 30
	}
	// defaults for bool pointers
	if cfg.Logging.Compress == nil {
		v := true
		cfg.Logging.Compress = &v
	}
	if cfg.Logging.Stdout == nil {
		v := true
		cfg.Logging.Stdout = &v
	}
	return &cfg, nil
}
