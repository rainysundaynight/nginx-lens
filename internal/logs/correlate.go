package logs

import (
	"sort"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

// ---------- Корреляция access + error ----------
// Сводка по upstream: 502% + connect_failed по логическому имени.

// UpstreamCorrelation — связанные метрики access и error.log.
type UpstreamCorrelation struct {
	Upstream       string   `json:"upstream"`
	Locations      []string `json:"locations,omitempty"`
	AccessRequests int      `json:"access_requests"`
	Access5xxPct   float64  `json:"access_5xx_pct"`
	Access502      int      `json:"access_502"`
	ErrorConnect   int      `json:"error_connect_failed"`
	ErrorTimeout   int      `json:"error_timeout"`
	ErrorTotal     int      `json:"error_total"`
}

// BuildCorrelations строит отчёт access + error по логическим upstream.
func BuildCorrelations(
	access *AccessSnapshot,
	errors *ErrorStats,
	graph map[string][]analyzer.BlastRadiusEntry,
	upstreams map[string][]string,
	streamGraph upstream.StreamGraph,
) []UpstreamCorrelation {
	index := BuildUpstreamIndex(upstreams)
	names := collectLogicalNames(upstreams, index, access, errors, graph)
	byName := make(map[string]*UpstreamCorrelation, len(names))

	for _, name := range names {
		c := &UpstreamCorrelation{Upstream: name}
		if graph != nil {
			seen := make(map[string]struct{})
			for _, e := range graph[name] {
				loc := e.Location
				if loc == "" {
					loc = "/"
				}
				if _, ok := seen[loc]; ok {
					continue
				}
				seen[loc] = struct{}{}
				c.Locations = append(c.Locations, loc)
			}
		}
		if streamGraph != nil {
			seen := make(map[string]struct{})
			for _, e := range streamGraph[name] {
				loc := "listen:" + e.Listen
				if loc == "listen:" {
					loc = "stream:" + name
				}
				if _, ok := seen[loc]; ok {
					continue
				}
				seen[loc] = struct{}{}
				c.Locations = append(c.Locations, loc)
			}
		}
		byName[name] = c
	}

	type accessAgg struct{ req, s5, s502 int }
	accessByName := make(map[string]*accessAgg)
	if access != nil && access.ByUpstream != nil {
		for k, v := range access.ByUpstream {
			name := ResolveLogicalUpstream(k, index)
			a := accessByName[name]
			if a == nil {
				a = &accessAgg{}
				accessByName[name] = a
			}
			a.req += v.Requests
			a.s5 += v.Status5xx
			a.s502 += v.Status502
		}
	}
	for name, a := range accessByName {
		c, ok := byName[name]
		if !ok {
			c = &UpstreamCorrelation{Upstream: name}
			byName[name] = c
		}
		c.AccessRequests = a.req
		c.Access502 = a.s502
		if a.req > 0 {
			c.Access5xxPct = float64(a.s5) / float64(a.req) * 100
		}
	}

	if errors != nil {
		for _, e := range errors.UpstreamErrors {
			name := ResolveLogicalUpstream(e.Upstream, index)
			c, ok := byName[name]
			if !ok {
				c = &UpstreamCorrelation{Upstream: name}
				byName[name] = c
			}
			c.ErrorTotal += e.Count
		}
		for raw, cnt := range errors.UpstreamConnect {
			name := ResolveLogicalUpstream(raw, index)
			c, ok := byName[name]
			if !ok {
				c = &UpstreamCorrelation{Upstream: name}
				byName[name] = c
			}
			c.ErrorConnect += cnt
		}
		for raw, cnt := range errors.UpstreamTimeout {
			name := ResolveLogicalUpstream(raw, index)
			c, ok := byName[name]
			if !ok {
				c = &UpstreamCorrelation{Upstream: name}
				byName[name] = c
			}
			c.ErrorTimeout += cnt
		}
	}

	out := make([]UpstreamCorrelation, 0, len(byName))
	for _, c := range byName {
		if c.Access5xxPct > 0 {
			c.Access5xxPct = float64(int(c.Access5xxPct*10+0.5)) / 10
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Upstream < out[j].Upstream })
	return out
}

func collectLogicalNames(upstreams map[string][]string, index map[string]string, access *AccessSnapshot, errors *ErrorStats, graph map[string][]analyzer.BlastRadiusEntry) []string {
	seen := make(map[string]struct{})
	var names []string
	add := func(n string) {
		n = strings.TrimSpace(n)
		if n == "" || n == "_direct" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	for name := range upstreams {
		add(name)
	}
	for n := range graph {
		add(n)
	}
	if access != nil {
		for k := range access.ByUpstream {
			add(ResolveLogicalUpstream(k, index))
		}
	}
	if errors != nil {
		for _, e := range errors.UpstreamErrors {
			add(ResolveLogicalUpstream(e.Upstream, index))
		}
	}
	return names
}
