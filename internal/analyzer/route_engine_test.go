package analyzer

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func TestExplainRouteExactLocation(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Directive: "server_name", Args: "example.com"},
			{Directive: "listen", Args: "80"},
			{Block: "location", LocModifier: "=", Arg: "/api", Directives: []parser.Node{
				{Directive: "proxy_pass", Args: "http://api_backend"},
			}},
			{Block: "location", Arg: "/", Directives: []parser.Node{
				{Directive: "root", Args: "/var/www"},
			}},
		},
	}}, nil)

	ex := ExplainRoute(tree, "http://example.com/api")
	if ex == nil || ex.Location == nil {
		t.Fatal("ожидался matched location")
	}
	if ex.Location.LocModifier != "=" || ex.Location.Arg != "/api" {
		t.Fatalf("location = %+v", ex.Location)
	}
	if ex.Upstream != "api_backend" && ex.ProxyPass != "http://api_backend" {
		t.Fatalf("proxy = %s upstream = %s", ex.ProxyPass, ex.Upstream)
	}
}

func TestExplainRoutePrefixLongest(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Directive: "server_name", Args: "example.com"},
			{Directive: "listen", Args: "80"},
			{Block: "location", Arg: "/api", Directives: []parser.Node{{Directive: "return", Args: "200 A"}}},
			{Block: "location", Arg: "/api/v2", Directives: []parser.Node{{Directive: "return", Args: "200 B"}}},
		},
	}}, nil)

	ex := ExplainRoute(tree, "http://example.com/api/v2/users")
	if ex == nil || ex.Location == nil || ex.Location.Arg != "/api/v2" {
		t.Fatalf("ожидался /api/v2, got %+v", ex)
	}
}

func TestExplainRouteRewriteTrace(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Directive: "server_name", Args: "example.com"},
			{Directive: "listen", Args: "80"},
			{Block: "location", Arg: "/api", Directives: []parser.Node{
				{Directive: "rewrite", Args: "^/api/(.*)$ /v1/$1 break"},
				{Directive: "proxy_pass", Args: "http://backend"},
			}},
		},
	}}, nil)
	ex := ExplainRoute(tree, "http://example.com/api/users")
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
