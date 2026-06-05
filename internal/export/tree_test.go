package export

import (
	"strings"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func sampleNodes() []parser.Node {
	return []parser.Node{
		{
			Block: "server",
			Arg:   "example.com",
			Directives: []parser.Node{
				{Directive: "listen", Args: "80"},
				{Block: "location", Arg: "/", Directives: []parser.Node{
					{Directive: "root", Args: "/var/www"},
				}},
			},
		},
		{
			Upstream: "api_backend",
			Servers:  []string{"127.0.0.1:8080"},
		},
	}
}

func TestRenderTreeMarkdown(t *testing.T) {
	out := RenderTreeMarkdown(sampleNodes(), 0)
	if !strings.Contains(out, "server example.com") {
		t.Fatalf("unexpected markdown: %q", out)
	}
	if !strings.Contains(out, "upstream api_backend") {
		t.Fatalf("missing upstream: %q", out)
	}
}

func TestRenderTreeHTML(t *testing.T) {
	out := RenderTreeHTML(sampleNodes())
	if !strings.Contains(out, "<ul>") || !strings.Contains(out, "server") {
		t.Fatalf("unexpected html: %q", out)
	}
}

func TestRenderTreeText(t *testing.T) {
	out := RenderTreeText(sampleNodes(), "", true)
	if !strings.Contains(out, "server example.com") {
		t.Fatalf("unexpected text tree: %q", out)
	}
}
