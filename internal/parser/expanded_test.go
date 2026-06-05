package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func countServerBlocks(tree *ConfigTree) int {
	n := 0
	var walk func([]Node)
	walk = func(nodes []Node) {
		for _, node := range nodes {
			if node.Block == "server" {
				n++
			}
			if len(node.Directives) > 0 {
				walk(node.Directives)
			}
		}
	}
	walk(tree.Directives)
	return n
}

func TestExpandedSkipsIncludeDirectives(t *testing.T) {
	dir := t.TempDir()
	confd := filepath.Join(dir, "conf.d")
	if err := os.MkdirAll(confd, 0o755); err != nil {
		t.Fatal(err)
	}
	nginxConf := filepath.Join(dir, "nginx.conf")
	testConf := filepath.Join(confd, "001-test.conf")
	if err := os.WriteFile(nginxConf, []byte("http { include conf.d/*.conf; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testConf, []byte("server { listen 80; location / { return 200; } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := "# configuration file " + nginxConf + ":\n" +
		"http {\n    include " + confd + "/*.conf;\n}\n" +
		"# configuration file " + testConf + ":\n" +
		"server {\n    listen 80;\n    location / { return 200; }\n}\n"
	tree, err := ParseExpandedOutput(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := countServerBlocks(tree); got != 1 {
		t.Fatalf("server blocks: got %d want 1", got)
	}
}
