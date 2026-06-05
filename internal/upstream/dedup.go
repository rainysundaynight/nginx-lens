package upstream

import "strings"

// ---------- Дедупликация upstream ----------
// Убирает повторы после merge nginx -T и dual-stack server-блоков.

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

// DedupeRefs убирает дубли ссылок на upstream (location/server/listen).
func DedupeRefs(refs []UpstreamRef) []UpstreamRef {
	seen := make(map[string]struct{}, len(refs))
	out := make([]UpstreamRef, 0, len(refs))
	for _, r := range refs {
		key := strings.Join([]string{
			r.UpstreamName,
			r.Location,
			r.FromDirective,
			r.Value,
			r.ConfigFile,
		}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
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
