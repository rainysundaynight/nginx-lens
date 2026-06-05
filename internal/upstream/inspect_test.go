package upstream

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func loadMinimalTree(t *testing.T) *parser.ConfigTree {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "testdata", "nginx", "minimal.conf")
	tree, err := parser.ParseConfig(path, parser.ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestFindUpstreamBlocks(t *testing.T) {
	tree := loadMinimalTree(t)
	blocks := FindUpstreamBlocks(tree)
	if len(blocks) != 1 || blocks[0].Name != "api_backend" {
		t.Fatalf("blocks=%v", blocks)
	}
}

func TestCollectKnownUpstreamNames(t *testing.T) {
	tree := loadMinimalTree(t)
	names := CollectKnownUpstreamNames(tree)
	if _, ok := names["api_backend"]; !ok {
		t.Fatalf("names=%v", names)
	}
}

func TestFindUpstreamReferences(t *testing.T) {
	tree := loadMinimalTree(t)
	names := CollectKnownUpstreamNames(tree)
	refs := FindUpstreamReferences(tree, names)
	if len(refs) == 0 {
		t.Fatal("expected proxy_pass reference")
	}
}

func TestExtractUpstreamName(t *testing.T) {
	if got := extractUpstreamName("http://api_backend"); got != "api_backend" {
		t.Fatalf("got %q", got)
	}
}
