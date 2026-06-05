package upstream

import "testing"

func TestDedupeServers(t *testing.T) {
	got := DedupeServers([]string{"127.0.0.1:80", "127.0.0.1:80", "10.0.0.1:80"})
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestDedupeRefs(t *testing.T) {
	refs := []UpstreamRef{
		{UpstreamName: "test", Location: "/", Listen: "80", Value: "http://test"},
		{UpstreamName: "test", Location: "/", Listen: "[::]:80", Value: "http://test"},
		{UpstreamName: "test", Location: "/", Listen: "80", Value: "http://test"},
	}
	got := DedupeRefs(refs)
	if len(got) != 1 {
		t.Fatalf("got %d want 1: %+v", len(got), got)
	}
}
