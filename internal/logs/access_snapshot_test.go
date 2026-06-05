package logs

import (
	"testing"
)

func TestComputeAccessSnapshotDirectByPath(t *testing.T) {
	lines := []LogLine{
		{Path: "/", Status: 502},
		{Path: "/", Status: 503},
		{Path: "/api", Status: 200, Upstream: "127.0.0.1:8080"},
	}
	snap := computeAccessSnapshot(lines)
	if snap.ByPath["/"].Requests != 2 || snap.ByPath["/"].Status5xx != 2 {
		t.Fatalf("by_path[/]=%+v", snap.ByPath["/"])
	}
	if _, ok := snap.ByPath["/api"]; ok {
		t.Fatal("upstream request should not be in by_path")
	}
	if snap.ByUpstream["_direct"].Requests != 2 {
		t.Fatalf("_direct=%+v", snap.ByUpstream["_direct"])
	}
}
