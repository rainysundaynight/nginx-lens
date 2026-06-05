package nginxinfo

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func TestCheckDirectiveModulesBuiltinProxy(t *testing.T) {
	info, err := ParseVersionOutput(sampleNginxV)
	if err != nil {
		t.Fatal(err)
	}
	tree := parser.NewConfigTree([]parser.Node{
		{
			Block: "server",
			Directives: []parser.Node{
				{
					Block: "location", Arg: "/",
					Directives: []parser.Node{{Directive: "proxy_pass", Args: "http://backend"}},
				},
			},
		},
	}, nil)
	issues := CheckDirectiveModules(tree, info)
	for _, iss := range issues {
		if iss.Module == "http_proxy" {
			t.Fatalf("http_proxy should be built-in, got issue: %+v", iss)
		}
	}
}

func TestCheckDirectiveModulesOptionalSSL(t *testing.T) {
	raw := `nginx version: nginx/1.24.0
configure arguments: --prefix=/etc/nginx
`
	info, err := ParseVersionOutput(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := parser.NewConfigTree([]parser.Node{
		{
			Block: "server",
			Directives: []parser.Node{
				{Directive: "ssl_certificate", Args: "/etc/ssl/cert.pem"},
			},
		},
	}, nil)
	issues := CheckDirectiveModules(tree, info)
	found := false
	for _, iss := range issues {
		if iss.Module == "http_ssl" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected http_ssl issue when module not in configure args")
	}
}

func TestCheckDirectiveModulesDisabledProxy(t *testing.T) {
	raw := `nginx version: nginx/1.24.0
configure arguments: --prefix=/etc/nginx --without-http_proxy_module
`
	info, err := ParseVersionOutput(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := parser.NewConfigTree([]parser.Node{
		{
			Block: "server",
			Directives: []parser.Node{
				{
					Block: "location", Arg: "/",
					Directives: []parser.Node{{Directive: "proxy_pass", Args: "http://backend"}},
				},
			},
		},
	}, nil)
	issues := CheckDirectiveModules(tree, info)
	found := false
	for _, iss := range issues {
		if iss.Module == "http_proxy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected http_proxy issue when explicitly disabled in configure args")
	}
}
