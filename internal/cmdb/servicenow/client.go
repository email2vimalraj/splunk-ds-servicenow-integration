package servicenow

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/example/splunk-ds-camr/internal/cmdb"
	"github.com/example/splunk-ds-camr/internal/config"
)

type Client struct {
	baseURL   string
	table     string
	query     string
	hostField string
	laneField string
	pageSize  int
	client    *http.Client
	auth      config.ServiceNowAuth
}

type response struct {
	Result []map[string]any `json:"result"`
	Offset int              `json:"_offset"`
	Total  int              `json:"total"`
}

func New(cfg config.ServiceNowConfig) *Client {
	tr := &http.Transport{}
	if cfg.InsecureSkipVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // for labs only
	}
	return &Client{
		baseURL:   cfg.BaseURL,
		table:     cfg.Table,
		query:     cfg.Query,
		hostField: ifEmpty(cfg.HostnameField, "host_name"),
		laneField: ifEmpty(cfg.LaneField, "business_service_lane"),
		pageSize:  cfg.PageSize,
		client:    &http.Client{Transport: tr, Timeout: cfg.Timeout.Duration},
		auth:      cfg.Auth,
	}
}

func ifEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (c *Client) Fetch(ctx context.Context) ([]cmdb.Entry, error) {
	var out []cmdb.Entry
	start := 0
	for {
		batch, total, err := c.fetchPage(ctx, start)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		start += len(batch)
		if start >= total || len(batch) == 0 {
			break
		}
	}
	return out, nil
}

func (c *Client) fetchPage(ctx context.Context, offset int) ([]cmdb.Entry, int, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, 0, err
	}
	u.Path = fmt.Sprintf("/api/now/table/%s", c.table)
	q := u.Query()
	if c.query != "" {
		q.Set("sysparm_query", c.query)
	}
	q.Set("sysparm_offset", fmt.Sprint(offset))
	q.Set("sysparm_limit", fmt.Sprint(c.pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if c.auth.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.auth.BearerToken)
	} else if c.auth.Username != "" {
		req.SetBasicAuth(c.auth.Username, c.auth.Password)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("servicenow http %d", resp.StatusCode)
	}
	var r response
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&r); err != nil {
		return nil, 0, err
	}
	var out []cmdb.Entry
	for _, row := range r.Result {
		h, _ := row[c.hostField].(string)
		lane, _ := row[c.laneField].(string)
		if h == "" || lane == "" {
			continue
		}
		out = append(out, cmdb.Entry{Hostname: h, BusinessServiceLane: lane})
	}
	// NOTE: ServiceNow API variations may use total count in headers; we use len+offset fallback
	total := r.Total
	if total == 0 {
		total = offset + len(out)
	}
	return out, total, nil
}

// Allow the package to compile on older Go versions where any import might be unused
var _ = time.Now
