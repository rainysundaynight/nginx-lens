package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/rainysundaynight/nginx-lens/internal/policy"
)

const sampleConf = `
upstream api_backend {
    server 127.0.0.1:8080;
}
server {
    listen 80;
    server_name example.com;
    location = /api {
        rewrite ^/api/(.*)$ /v1/$1 break;
        proxy_pass http://api_backend;
    }
    location / {
        root /var/www;
    }
}
`

func TestAnalysisPipeline(t *testing.T) {
	path := writeConf(t, sampleConf)
	tree, err := parser.ParseConfig(path, parser.ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	result := analyzer.RunAnalysis(tree)
	issues := analyzer.CollectIssues(result)
	if len(issues) == 0 {
		t.Log("issues пуст — допустимо для минимального конфига")
	}
	graph := analyzer.BuildDependencyGraph(tree)
	if len(graph["api_backend"]) == 0 {
		t.Fatal("ожидалась ссылка на api_backend в dependency graph")
	}
	ex := analyzer.ExplainRoute(tree, "http://example.com/api")
	if ex == nil || ex.Location == nil {
		t.Fatal("explain должен найти location")
	}
	hasRewrite := false
	for _, step := range ex.Trace {
		if step.Step == "rewrite" {
			hasRewrite = true
		}
	}
	if !hasRewrite {
		t.Fatal("trace должен содержать rewrite")
	}
}

func TestPolicyAndScore(t *testing.T) {
	path := writeConf(t, sampleConf+`
server {
    listen 443 ssl;
    server_name secure.example.com;
    autoindex on;
    ssl_protocols TLSv1 TLSv1.2;
}
`)
	tree, err := parser.ParseNginxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	engine := policy.NewEngine([]string{"security-baseline", "mozilla-ssl"}, nil)
	policyIssues := engine.Run(tree)
	if len(policyIssues) == 0 {
		t.Fatal("policy engine должен найти нарушения")
	}
	result := analyzer.RunAnalysis(tree)
	score := analyzer.ComputeScore(result, len(policyIssues), 0)
	if score.Total <= 0 || score.Total > 100 {
		t.Fatalf("score вне диапазона: %v", score.Total)
	}
}

func TestConfigSchema(t *testing.T) {
	cfg := config.DefaultConfig()
	errs := config.ValidateSchema(&cfg)
	if len(errs) != 0 {
		t.Fatalf("default config invalid: %v", errs)
	}
}

func writeConf(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
