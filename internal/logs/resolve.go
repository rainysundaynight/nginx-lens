package logs

import (
	"net/url"
	"strings"
)

// ---------- Резолв upstream: логический ↔ адрес ----------
// Сопоставление имён upstream с host:port из access/error.log.

// BuildUpstreamIndex строит индекс адрес → логическое имя upstream.
func BuildUpstreamIndex(upstreams map[string][]string) map[string]string {
	idx := make(map[string]string)
	for name, servers := range upstreams {
		idx[strings.ToLower(name)] = name
		for _, srv := range servers {
			for _, key := range addressKeys(srv) {
				idx[key] = name
			}
		}
	}
	return idx
}

// ResolveLogicalUpstream возвращает логическое имя upstream по ключу из лога.
func ResolveLogicalUpstream(logKey string, index map[string]string) string {
	logKey = strings.TrimSpace(logKey)
	if logKey == "" {
		return ""
	}
	if name, ok := index[strings.ToLower(logKey)]; ok {
		return name
	}
	for _, key := range addressKeys(logKey) {
		if name, ok := index[key]; ok {
			return name
		}
	}
	return logKey
}

func addressKeys(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	keys := map[string]struct{}{}
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" {
			keys[s] = struct{}{}
		}
	}
	add(raw)
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		add(u.Host)
		add(strings.TrimPrefix(u.Host, "http://"))
		add(strings.TrimPrefix(u.Host, "https://"))
	}
	stripped := raw
	for _, p := range []string{"http://", "https://", "grpc://"} {
		stripped = strings.TrimPrefix(stripped, p)
	}
	add(stripped)
	if h, p, ok := strings.Cut(stripped, ":"); ok {
		add(h + ":" + p)
		add(h)
	}
	var out []string
	for k := range keys {
		out = append(out, k)
	}
	return out
}
