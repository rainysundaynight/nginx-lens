package policy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func loadFixture(t *testing.T, name string) *parser.ConfigTree {
	t.Helper()
	path := filepath.Join(repoRoot(t), "testdata", "nginx", name)
	tree, err := parser.ParseConfig(path, parser.ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestSecurityBaselinePack(t *testing.T) {
	tree := loadFixture(t, "ssl-bad.conf")
	engine := NewEngine([]string{"security-baseline"}, nil)
	issues := engine.Run(tree)
	if len(issues) == 0 {
		t.Fatal("security-baseline должен найти autoindex on")
	}
	found := false
	for _, iss := range issues {
		if iss.RuleID == "no_autoindex" {
			found = true
		}
	}
	if !found {
		t.Fatal("ожидалось правило no_autoindex")
	}
}

func TestMozillaSSLPack(t *testing.T) {
	tree := loadFixture(t, "ssl-bad.conf")
	engine := NewEngine([]string{"mozilla-ssl"}, nil)
	issues := engine.Run(tree)
	if len(issues) == 0 {
		t.Fatal("mozilla-ssl должен найти TLS 1.0")
	}
}

func TestPerformanceBaselinePack(t *testing.T) {
	tree := loadFixture(t, "minimal.conf")
	engine := NewEngine([]string{"performance-baseline"}, nil)
	issues := engine.Run(tree)
	if len(issues) == 0 {
		t.Fatal("performance-baseline должен найти upstream без keepalive")
	}
}

func TestCachingPack(t *testing.T) {
	conf := `
server {
    listen 80;
    location /static/ {
        root /var/www;
    }
    location /api {
        proxy_cache my_cache;
        proxy_pass http://127.0.0.1:8080;
    }
}
`
	path := writeTempConf(t, conf)
	tree, err := parser.ParseNginxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine([]string{"caching"}, nil)
	issues := engine.Run(tree)
	if len(issues) == 0 {
		t.Fatal("caching pack должен найти нарушения")
	}
}

func TestRateLimitPack(t *testing.T) {
	conf := `
server {
    listen 80;
    location / {
        limit_req zone=api burst=5;
        proxy_pass http://127.0.0.1:8080;
    }
}
`
	path := writeTempConf(t, conf)
	tree, err := parser.ParseNginxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	engine := NewEngine([]string{"rate-limit"}, nil)
	issues := engine.Run(tree)
	if len(issues) == 0 {
		t.Fatal("rate-limit pack должен найти limit_req")
	}
}

func TestCustomRule(t *testing.T) {
	tree := loadFixture(t, "minimal.conf")
	engine := NewEngine(nil, []CustomRule{
		{
			ID:       "custom-proxy-pass",
			Match:    "directive.proxy_pass",
			Severity: "medium",
			Message:  "найден proxy_pass",
		},
	})
	issues := engine.Run(tree)
	if len(issues) == 0 {
		t.Fatal("custom rule должен сработать")
	}
	if issues[0].Severity != analyzer.SeverityMedium {
		t.Fatalf("severity=%s, want medium", issues[0].Severity)
	}
}

func TestUnknownPackReturnsEmpty(t *testing.T) {
	tree := loadFixture(t, "minimal.conf")
	engine := NewEngine([]string{"totally-unknown-pack"}, nil)
	issues := engine.Run(tree)
	if len(issues) != 0 {
		t.Fatalf("неизвестный pack не должен давать issues: %v", issues)
	}
}

func writeTempConf(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
