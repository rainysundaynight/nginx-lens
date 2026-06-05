package parser

import "strings"

// ---------- Дедупликация upstream-серверов ----------
// Убирает повторы после merge секций nginx -T.

// DedupeServers оставляет уникальные адреса server в порядке появления.
func DedupeServers(servers []string) []string {
	seen := make(map[string]struct{}, len(servers))
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		key := strings.TrimSpace(s)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// DedupeUpstreamMap дедуплицирует серверы во всех upstream.
func DedupeUpstreamMap(upstreams map[string][]string) map[string][]string {
	out := make(map[string][]string, len(upstreams))
	for name, servers := range upstreams {
		out[name] = DedupeServers(servers)
	}
	return out
}
