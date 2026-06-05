package analyzer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func loadFixtureTree(t *testing.T, name string) *parser.ConfigTree {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "testdata", "nginx", name)
	tree, err := parser.ParseConfig(path, parser.ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestRunAnalysisMinimal(t *testing.T) {
	tree := loadFixtureTree(t, "minimal.conf")
	result := RunAnalysis(tree)
	if len(result.LocationConflicts) != 0 {
		t.Fatalf("unexpected location conflicts for / and =/api: %v", result.LocationConflicts)
	}
	treeNested := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Block: "location", Arg: "/api"},
			{Block: "location", Arg: "/api/v1"},
		},
	}}, nil)
	if len(FindLocationConflicts(treeNested)) == 0 {
		t.Fatal("expected /api vs /api/v1 conflict")
	}
}

func TestFindDuplicateDirectives(t *testing.T) {
	conf := `
server {
    listen 80;
    listen 80;
    server_name dup.example.com;
}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	if err := os.WriteFile(path, []byte(conf), 0644); err != nil {
		t.Fatal(err)
	}
	tree, err := parser.ParseNginxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	dups := FindDuplicateDirectives(tree)
	if len(dups) == 0 {
		t.Fatal("expected duplicate listen")
	}
}

func TestFindEmptyBlocks(t *testing.T) {
	conf := `
server {
    listen 80;
    location /empty {
    }
}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	if err := os.WriteFile(path, []byte(conf), 0644); err != nil {
		t.Fatal(err)
	}
	tree, err := parser.ParseNginxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	empty := FindEmptyBlocks(tree)
	if len(empty) == 0 {
		t.Fatal("expected empty location block")
	}
}

func TestComputeScore(t *testing.T) {
	tree := loadFixtureTree(t, "minimal.conf")
	result := RunAnalysis(tree)
	report := ComputeScoreFromIssues(CollectIssues(result), 0)
	if report.Total <= 0 || report.Total > 100 {
		t.Fatalf("score=%v", report.Total)
	}
	if len(report.Categories) == 0 {
		t.Fatal("expected categories")
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	tree := loadFixtureTree(t, "minimal.conf")
	graph := BuildDependencyGraph(tree)
	if len(graph["api_backend"]) == 0 {
		t.Fatal("expected dependency on api_backend")
	}
}

func TestShouldIncludeIssueFilter(t *testing.T) {
	opts := FilterOptions{MinSeverity: SeverityHigh, SkipTypes: map[string]struct{}{"x": {}}}
	if ShouldIncludeIssue("x", SeverityHigh, opts) {
		t.Fatal("skip type should exclude issue")
	}
	if !ShouldIncludeIssue("y", SeverityHigh, opts) {
		t.Fatal("high severity should pass")
	}
	if ShouldIncludeIssue("y", SeverityLow, opts) {
		t.Fatal("low severity should be filtered")
	}
}