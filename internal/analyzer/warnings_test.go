package analyzer

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func TestSSLProtocolsWeakFalsePositive(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "http", Directives: []parser.Node{{Directive: "ssl_protocols", Args: "TLSv1.2 TLSv1.3"}},
	}}, nil)
	for _, w := range FindWarnings(tree) {
		if w.Type == "ssl_protocols_weak" {
			t.Fatalf("TLSv1.2 TLSv1.3 must not trigger ssl_protocols_weak")
		}
	}
}

func TestListen8443NoFalsePositive(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server", Directives: []parser.Node{{Directive: "listen", Args: "8443"}},
	}}, nil)
	for _, w := range FindWarnings(tree) {
		if w.Type == "listen_443_no_ssl" || w.Type == "listen_443_no_http2" {
			t.Fatalf("8443 false positive: %s", w.Type)
		}
	}
}

func TestListen443SSLNoFalsePositiveHTTP2(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server", Directives: []parser.Node{{Directive: "listen", Args: "443 ssl"}},
	}}, nil)
	for _, w := range FindWarnings(tree) {
		if w.Type == "listen_443_no_ssl" {
			t.Fatalf("443 ssl should not trigger no_ssl")
		}
	}
}

func TestCipherExclusionNoFalsePositive(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server", Directives: []parser.Node{{Directive: "ssl_ciphers", Args: "HIGH:!MD5:!aNull"}},
	}}, nil)
	for _, w := range FindWarnings(tree) {
		if w.Type == "ssl_ciphers_weak" {
			t.Fatalf("cipher exclusion false positive: %s", w.Value)
		}
	}
}

func TestServerTokensOffMissing(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server", Directives: []parser.Node{{Directive: "listen", Args: "80"}},
	}}, nil)
	found := false
	for _, w := range FindWarnings(tree) {
		if w.Type == "server_tokens_off_missing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected server_tokens_off_missing")
	}
}

func TestServerTokensOffAtHTTP(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "http",
		Directives: []parser.Node{
			{Directive: "server_tokens", Args: "off"},
			{Block: "server", Directives: []parser.Node{{Directive: "listen", Args: "80"}}},
		},
	}}, nil)
	for _, w := range FindWarnings(tree) {
		if w.Type == "server_tokens_off_missing" {
			t.Fatal("http server_tokens off should suppress warning")
		}
	}
}

func TestMissingSecurityHeaderOnlySSL(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server", Directives: []parser.Node{{Directive: "listen", Args: "80"}},
	}}, nil)
	for _, w := range FindWarnings(tree) {
		if w.Type == "missing_security_header" {
			t.Fatal("HTTP-only server should not require security headers")
		}
	}
}

func TestNormalizeListenKeyConflict(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{
		{
			Block: "server", File: "a.conf", Line: 1,
			Directives: []parser.Node{
				{Directive: "listen", Args: "80"},
				{Directive: "server_name", Args: "example.com"},
			},
		},
		{
			Block: "server", File: "b.conf", Line: 10,
			Directives: []parser.Node{
				{Directive: "listen", Args: "[::]:80"},
				{Directive: "server_name", Args: "example.com"},
			},
		},
	}, nil)
	conflicts := FindListenServerNameConflicts(tree)
	if len(conflicts) == 0 {
		t.Fatal("expected conflict for 80 and [::]:80 with same server_name")
	}
}

func TestLocationRootVsAPI(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Block: "location", Arg: "/"},
			{Block: "location", Arg: "/api"},
		},
	}}, nil)
	if len(FindLocationConflicts(tree)) != 0 {
		t.Fatal("/ vs /api should not be flagged as conflict")
	}
}

func TestHasWeakTLSProtocols(t *testing.T) {
	if !HasWeakTLSProtocols("TLSv1 TLSv1.2") {
		t.Fatal("expected weak")
	}
	if HasWeakTLSProtocols("TLSv1.2 TLSv1.3") {
		t.Fatal("expected ok")
	}
}

func TestHostMatchesWildcardApex(t *testing.T) {
	if !hostMatchesWildcard("example.com", "*.example.com") {
		t.Fatal("apex should match *.example.com")
	}
}
