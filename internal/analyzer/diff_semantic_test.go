package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func TestDiffSemanticUpstream(t *testing.T) {
	dir := t.TempDir()
	cfg1 := filepath.Join(dir, "a.conf")
	cfg2 := filepath.Join(dir, "b.conf")
	writeFile(t, cfg1, `
upstream api { server 10.0.0.1:8080; }
server {
    listen 80;
    server_name example.com;
    location /api { proxy_pass http://api; }
}
`)
	writeFile(t, cfg2, `
upstream api { server 10.0.0.2:8080; }
server {
    listen 80;
    server_name example.com;
    location /api { proxy_pass http://api; }
}
`)
	t1, err := parser.ParseNginxConfig(cfg1)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := parser.ParseNginxConfig(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	diffs := DiffSemantic(t1, t2)
	if len(diffs) == 0 {
		t.Fatal("expected upstream diff")
	}
	found := false
	for _, d := range diffs {
		if d.Type == "changed" && len(d.Path) >= 2 && d.Path[0] == "upstream" && d.Path[1] == "api" {
			found = true
		}
	}
	if !found {
		t.Fatalf("upstream change not found: %+v", diffs)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
