package integration

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

func TestSingleUpstreamNoDuplicates(t *testing.T) {
	path := writeConf(t, `
upstream test {
    server 127.0.0.1:80;
}
server {
    listen 80;
    location / {
        proxy_pass http://test;
    }
}
`)
	tree, err := parser.ParseConfig(path, parser.ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	names := upstream.CollectKnownUpstreamNames(tree)
	refs := upstream.FindUpstreamReferences(tree, names)
	if len(refs) != 1 {
		t.Fatalf("refs: got %d want 1: %+v", len(refs), refs)
	}
	ups := tree.GetUpstreams()
	if len(ups["test"]) != 1 {
		t.Fatalf("servers: got %v want 1", ups["test"])
	}
	graph := analyzer.BuildDependencyGraph(tree)
	if len(graph["test"]) != 1 {
		t.Fatalf("graph: got %d want 1: %+v", len(graph["test"]), graph["test"])
	}
}

func TestDualStackServerDedupesLocation(t *testing.T) {
	path := writeConf(t, `
upstream test {
    server 127.0.0.1:80;
}
server {
    listen 80;
    location / {
        proxy_pass http://test;
    }
}
server {
    listen [::]:80;
    location / {
        proxy_pass http://test;
    }
}
`)
	tree, err := parser.ParseConfig(path, parser.ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	graph := analyzer.BuildDependencyGraph(tree)
	if len(graph["test"]) != 1 {
		t.Fatalf("graph: got %d want 1: %+v", len(graph["test"]), graph["test"])
	}
}

func TestExpandedMergeDedupesServers(t *testing.T) {
	out := `# configuration file /etc/nginx/a.conf:
upstream test {
    server 127.0.0.1:80;
}
# configuration file /etc/nginx/b.conf:
upstream test {
    server 127.0.0.1:80;
}
`
	tree, err := parser.ParseExpandedOutput(out)
	if err != nil {
		t.Fatal(err)
	}
	ups := tree.GetUpstreams()
	if len(ups["test"]) != 1 {
		t.Fatalf("servers: got %v want 1", ups["test"])
	}
}
