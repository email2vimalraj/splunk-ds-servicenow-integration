# Splunk Deployment Server - ServiceNow Integration

A Go utility for Splunk Deployment Servers to map CMDB server entries to destinations and update `serverclass.conf` whitelists with compressed wildcard patterns.

## Features

- Fetch servers from CMDB (dummy adapter; ServiceNow support)
- Configurable refresh interval loop
- Map destinations to one or more Business Service Lanes (e.g., dest1 -> [lane1,lane2], dest2 -> [lane3,lane4])
- Compress hostnames to wildcard patterns (e.g., `abc001`,`abc002` -> `abc*`)
- Update `serverclass.conf` whitelist per server class/app

## Install

Requires Go 1.22+.

## Configure

Copy the example config and edit as needed:

```bash
cp config.example.yaml config.yaml
```

Optionally set CAMR_CONFIG to point to your config file.

## Run

```bash
go run ./cmd/splunk-ds-camr
```

Build a binary:

```bash
go build -o bin/splunk-ds-camr ./cmd/splunk-ds-camr
```

## Config file schema

See `config.example.yaml`. Key parts:

- `refreshInterval`: Go duration, e.g., `1m` or `5m`
- `dryRun`: if true, do not write serverclass.conf; just log what would change
- `destinations`: map destination -> array of lanes
- `cmdb.type`: `dummy` or `servicenow`
- `cmdb.dummy.entries`: list of hostname + businessServiceLane
- `cmdb.servicenow`: connection (baseURL, table, query, hostnameField, laneField, pageSize, timeout, auth)
- `serverclass.path`: location of Splunk `serverclass.conf`
- `serverclass.backup`: whether to create a `.bak` before writing
- `serverclass.appClass`: app -> serverClass name
- `serverclass.appDestination`: app -> destination key
- `serverclass.dryRunApps`: list of app names to treat as dry-run even when global dryRun is false
- `wildcard`: controls pattern generation
  - `mode`: `trailingOnly` (default) or `internalNumeric`
  - `minGroupSize`: minimum hosts required to emit a wildcard (default 2)
  - `requireMinFixedPrefix`: guardrail to avoid overly broad patterns (default 0)

## Dry-run overrides via env:

```bash
CAMR_DRY_RUN=1 go run ./cmd/splunk-ds-camr
```

## Notes

- The `serverclass.conf` writer preserves other sections but overwrites whitelist entries in the specified `serverClass:<name>` sections.
- Dry-run logs show per-app diffs: counts of additions/removals, without writing the file.
- ServiceNow queries use encoded query syntax; use bearer token or basic auth.
