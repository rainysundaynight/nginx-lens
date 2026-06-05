package logs

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

func TestBuildCorrelationsPerUpstreamErrors(t *testing.T) {
	access := &AccessSnapshot{
		ByUpstream: map[string]UpstreamAccess{
			"10.0.0.1:8080": {Requests: 100, Status5xx: 12, Status502: 5, ErrorPct: 12},
		},
	}
	errors := &ErrorStats{
		UpstreamErrors: []ErrorEntry{{Upstream: "http://10.0.0.1:8080", Count: 3}},
		UpstreamConnect: map[string]int{"http://10.0.0.1:8080": 2},
		ConnectFailed: 2,
	}
	ups := map[string][]string{"api_backend": {"10.0.0.1:8080"}}
	graph := map[string][]analyzer.BlastRadiusEntry{
		"api_backend": {{Location: "/api"}},
	}
	cors := BuildCorrelations(access, errors, graph, ups, nil)
	if len(cors) != 1 {
		t.Fatalf("expected 1 correlation, got %d", len(cors))
	}
	c := cors[0]
	if c.Upstream != "api_backend" {
		t.Fatalf("upstream=%s", c.Upstream)
	}
	if c.ErrorConnect != 2 {
		t.Fatalf("connect=%d want 2", c.ErrorConnect)
	}
	if c.ErrorTimeout != 0 {
		t.Fatalf("timeout=%d want 0", c.ErrorTimeout)
	}
	if c.Access5xxPct < 11.9 || c.Access5xxPct > 12.1 {
		t.Fatalf("5xx pct=%f", c.Access5xxPct)
	}
	if len(c.Locations) != 1 || c.Locations[0] != "/api" {
		t.Fatalf("locations=%v", c.Locations)
	}
}

func TestBuildCorrelationsDirectPath(t *testing.T) {
	access := &AccessSnapshot{
		TotalRequests: 383,
		Status5xx:     383,
		ByUpstream: map[string]UpstreamAccess{
			"_direct": {Requests: 383, Status5xx: 383, Status502: 383},
		},
		ByPath: map[string]PathAccess{
			"/": {Requests: 383, Status5xx: 383, Status502: 383},
		},
	}
	errors := &ErrorStats{
		UpstreamErrors: []ErrorEntry{{Upstream: "http://127.0.0.1:80/", Count: 1}},
	}
	ups := map[string][]string{"test": {"127.0.0.1:80"}}
	graph := map[string][]analyzer.BlastRadiusEntry{
		"test": {{Location: "/", ProxyPass: "http://test"}},
	}
	cors := BuildCorrelations(access, errors, graph, ups, nil)
	if len(cors) != 1 {
		t.Fatalf("expected 1 correlation, got %d", len(cors))
	}
	c := cors[0]
	if c.Upstream != "test" {
		t.Fatalf("upstream=%s", c.Upstream)
	}
	if c.AccessRequests != 383 || c.Access5xxPct < 99.9 {
		t.Fatalf("access=%d pct=%f", c.AccessRequests, c.Access5xxPct)
	}
	if c.ErrorTotal != 1 {
		t.Fatalf("errors=%d", c.ErrorTotal)
	}
}

func TestResolveLogicalUpstreamTrailingSlash(t *testing.T) {
	idx := BuildUpstreamIndex(map[string][]string{"test": {"127.0.0.1:80"}})
	if got := ResolveLogicalUpstream("http://127.0.0.1:80/", idx); got != "test" {
		t.Fatalf("got %q want test", got)
	}
}

func TestBuildCorrelationsStreamLocations(t *testing.T) {
	stream := upstream.StreamGraph{
		"tcp_up": {{Listen: "9000", UpstreamName: "tcp_up"}},
	}
	cors := BuildCorrelations(nil, nil, nil, map[string][]string{"tcp_up": {"127.0.0.1:9000"}}, stream)
	if len(cors) != 1 {
		t.Fatalf("got %d", len(cors))
	}
	if len(cors[0].Locations) != 1 || cors[0].Locations[0] != "listen:9000" {
		t.Fatalf("locations=%v", cors[0].Locations)
	}
}
