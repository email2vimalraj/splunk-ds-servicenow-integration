package patterns

import (
	"sort"
	"strings"
)

// GenerateWildcards compresses hostnames by finding common prefixes and replacing trailing numeric sequences with a '*'.
// It returns deterministic, sorted patterns.
func GenerateWildcards(hosts []string) []string {
	if len(hosts) == 0 {
		return nil
	}
	h := append([]string(nil), hosts...)
	sort.Strings(h)

	groups := map[string][]string{}
	for _, host := range h {
		pfx, num := splitNumericSuffix(host)
		key := pfx
		if num == "" {
			// no numeric suffix, group by full host
			key = host
		}
		groups[key] = append(groups[key], host)
	}

	var out []string
	for key, items := range groups {
		if len(items) >= 2 {
			// patternize
			out = append(out, key+"*")
		} else {
			out = append(out, items[0])
		}
	}
	sort.Strings(out)
	return dedupe(out)
}

func splitNumericSuffix(s string) (prefix, numeric string) {
	// find trailing digits
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	return s[:i], s[i:]
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return in
	}
	res := in[:0]
	var last string
	for _, v := range in {
		if v != last {
			res = append(res, v)
			last = v
		}
	}
	return res
}

// Options controls wildcard generation behavior.
type Options struct {
	// Mode: "trailingOnly" (default) or "internalNumeric"
	Mode string
	// MinGroupSize: minimum number of hosts required to emit a wildcard pattern
	MinGroupSize int
	// RequireMinFixedPrefix: minimum number of fixed characters before the first '*' when emitting patterns
	RequireMinFixedPrefix int
}

// GenerateWildcardsWithOptions supports advanced grouping, including internal numeric blocks.
func GenerateWildcardsWithOptions(hosts []string, opts Options) []string {
	if len(hosts) == 0 {
		return nil
	}
	if opts.Mode == "" {
		opts.Mode = "trailingOnly"
	}
	if opts.MinGroupSize <= 0 {
		opts.MinGroupSize = 2
	}
	if opts.RequireMinFixedPrefix < 0 {
		opts.RequireMinFixedPrefix = 0
	}

	switch opts.Mode {
	case "internalNumeric":
		return genInternalNumeric(hosts, opts)
	default:
		return GenerateWildcards(hosts)
	}
}

func genInternalNumeric(hosts []string, opts Options) []string {
	// Normalize by replacing digit runs with '#'
	groups := map[string][]string{}
	for _, s := range hosts {
		norm := normalizeDigits(s)
		groups[norm] = append(groups[norm], s)
	}
	var out []string
	for norm, items := range groups {
		sort.Strings(items)
		if len(items) >= opts.MinGroupSize {
			pat := patternFromNormalized(norm)
			if firstStar := strings.IndexByte(pat, '*'); firstStar == -1 || firstStar >= opts.RequireMinFixedPrefix {
				out = append(out, pat)
				continue
			}
			// Guardrail: fixed prefix too short, fall back to trailing-only compression for this subset
			out = append(out, GenerateWildcards(items)...)
			continue
		}
		// Below threshold: emit explicit hosts (no wildcard)
		out = append(out, items...)
	}
	sort.Strings(out)
	return dedupe(out)
}

// normalizeDigits replaces each contiguous decimal run with a single '#'.
func normalizeDigits(s string) string {
	b := strings.Builder{}
	inDigits := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			if !inDigits {
				b.WriteByte('#')
				inDigits = true
			}
		} else {
			inDigits = false
			b.WriteByte(c)
		}
	}
	return b.String()
}

func patternFromNormalized(norm string) string {
	return strings.ReplaceAll(norm, "#", "*")
}

// Match checks if a hostname matches a simple wildcard pattern with a single '*' at the end.
func Match(pattern, host string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(host, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == host
}
