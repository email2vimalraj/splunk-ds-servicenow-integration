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

// Match checks if a hostname matches a simple wildcard pattern with a single '*' at the end.
func Match(pattern, host string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(host, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == host
}
