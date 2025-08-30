package cmdb

import (
	"context"

	"github.com/example/splunk-ds-camr/internal/config"
)

type Entry struct {
	Hostname            string
	BusinessServiceLane string
}

type Client interface {
	Fetch(ctx context.Context) ([]Entry, error)
}

type dummyClient struct {
	entries []Entry
}

func NewDummy(cfg config.DummyCMDBConfig) Client {
	d := &dummyClient{}
	for _, e := range cfg.Entries {
		d.entries = append(d.entries, Entry{Hostname: e.Hostname, BusinessServiceLane: e.BusinessServiceLane})
	}
	return d
}

func (d *dummyClient) Fetch(ctx context.Context) ([]Entry, error) {
	return d.entries, nil
}
