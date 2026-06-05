package analyzer

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Предупреждения и best practices ----------
// SSL, security headers, limits, deprecated directives.

// Warning — найденная проблема в конфигурации.
type Warning struct {
	Type      string       `json:"type"`
	Directive string       `json:"directive"`
	Context   *parser.Node `json:"context,omitempty"`
	Value     string       `json:"value"`
}

var deprecatedDirectives = map[string]string{
	"ssl":               "ssl директива устарела, используйте listen ... ssl",
	"spdy":              "spdy устарел, используйте http2",
	"ssl_session_cache": "ssl_session_cache устарел в новых версиях",
}

var securityHeaders = []string{
	"X-Frame-Options",
	"Strict-Transport-Security",
	"X-Content-Type-Options",
	"Referrer-Policy",
	"Content-Security-Policy",
}

type limitRange struct{ min, max int64 }

var limits = map[string]limitRange{
	"client_max_body_size":    {1024 * 1024, 1024 * 1024 * 100},
	"proxy_buffer_size":       {4096, 1024 * 1024},
	"proxy_buffers":           {2, 32},
	"proxy_busy_buffers_size": {4096, 1024 * 1024},
}

// FindWarnings находит потенциально опасные директивы и нарушения best practices.
func FindWarnings(tree *parser.ConfigTree) []Warning {
	var warnings []Warning
	ctx := collectConfigContext(tree)
	sslHeaders := make(map[string]map[string]struct{})

	for _, item := range Walk(tree) {
		d := item.Node
		parent := item.Parent

		if d.Directive == "proxy_pass" {
			if !regexp.MustCompile(`^(http|https)://`).MatchString(d.Args) {
				warnings = append(warnings, Warning{Type: "proxy_pass_no_scheme", Directive: "proxy_pass", Context: parent, Value: d.Args})
			}
		}
		if d.Directive == "autoindex" && strings.TrimSpace(d.Args) == "on" {
			warnings = append(warnings, Warning{Type: "autoindex_on", Directive: "autoindex", Context: parent, Value: "on"})
		}
		if d.Block == "if" {
			warnings = append(warnings, Warning{Type: "if_block", Directive: "if", Context: parent, Value: ""})
		}
		if d.Block == "server" && !serverTokensEffectiveOff(d, ctx) {
			warnings = append(warnings, Warning{Type: "server_tokens_off_missing", Directive: "server_tokens", Context: &d, Value: "off"})
		}
		if d.Directive == "ssl_certificate" || d.Directive == "ssl_certificate_key" {
			if strings.TrimSpace(d.Args) == "" {
				warnings = append(warnings, Warning{Type: "ssl_missing", Directive: d.Directive, Context: parent, Value: ""})
			}
		}
		if d.Directive == "ssl_protocols" {
			if HasWeakTLSProtocols(d.Args) {
				warnings = append(warnings, Warning{Type: "ssl_protocols_weak", Directive: "ssl_protocols", Context: parent, Value: d.Args})
			}
		}
		if d.Directive == "ssl_ciphers" && HasWeakSSLCiphers(d.Args) {
			warnings = append(warnings, Warning{Type: "ssl_ciphers_weak", Directive: "ssl_ciphers", Context: parent, Value: d.Args})
		}
		if d.Directive == "listen" && ListenPortIs443(d.Args) && !ListenHasSSL(d.Args) {
			warnings = append(warnings, Warning{Type: "listen_443_no_ssl", Directive: "listen", Context: parent, Value: d.Args})
		}
		if d.Directive == "listen" && ListenPortIs443(d.Args) && ListenHasSSL(d.Args) && !strings.Contains(d.Args, "http2") {
			warnings = append(warnings, Warning{Type: "listen_443_no_http2", Directive: "listen", Context: parent, Value: d.Args})
		}
		if d.Block == "server" && serverHasPublicListen(d) {
			hasLimit := ctx.httpLimitReq || ctx.httpLimitConn
			if !hasLimit {
				for _, sub := range WalkNodes(d.Directives, &d) {
					if sub.Node.Directive == "limit_req" || sub.Node.Directive == "limit_conn" {
						hasLimit = true
						break
					}
				}
			}
			if !hasLimit {
				warnings = append(warnings, Warning{Type: "no_limit_req_conn", Directive: "server", Context: &d, Value: ""})
			}
		}
		if d.Directive == "add_header" {
			srv := findAncestorServer(item)
			if srv != nil && serverHasSSLListen(*srv) {
				key := ServerScopeKey(*srv)
				if sslHeaders[key] == nil {
					sslHeaders[key] = make(map[string]struct{})
				}
				for _, h := range securityHeaders {
					if strings.Contains(d.Args, h) {
						sslHeaders[key][h] = struct{}{}
					}
				}
			}
		}
		if msg, ok := deprecatedDirectives[d.Directive]; ok {
			warnings = append(warnings, Warning{Type: "deprecated", Directive: d.Directive, Context: parent, Value: msg})
		}
		for lim, rng := range limits {
			if d.Directive == lim {
				parts := strings.Fields(d.Args)
				if len(parts) > 0 {
					size := parseSize(parts[0])
					if size != nil {
						if *size < rng.min {
							warnings = append(warnings, Warning{Type: "limit_too_small", Directive: lim, Context: parent, Value: parts[0]})
						}
						if *size > rng.max {
							warnings = append(warnings, Warning{Type: "limit_too_large", Directive: lim, Context: parent, Value: parts[0]})
						}
					}
				}
			}
		}
	}
	for _, item := range Walk(tree) {
		if item.Node.Block != "server" || !serverHasSSLListen(item.Node) {
			continue
		}
		key := ServerScopeKey(item.Node)
		have := sslHeaders[key]
		for _, h := range securityHeaders {
			if _, ok := have[h]; !ok {
				warnings = append(warnings, Warning{Type: "missing_security_header", Directive: "add_header", Context: &item.Node, Value: h})
			}
		}
	}
	return warnings
}

func findAncestorServer(item WalkItem) *parser.Node {
	if item.Node.Block == "server" {
		return &item.Node
	}
	if item.Parent != nil && item.Parent.Block == "server" {
		return item.Parent
	}
	for i := len(item.Ancestors) - 1; i >= 0; i-- {
		if item.Ancestors[i] != nil && item.Ancestors[i].Block == "server" {
			return item.Ancestors[i]
		}
	}
	return nil
}

// HasWeakTLSProtocols проверяет TLS 1.0/1.1 в ssl_protocols.
func HasWeakTLSProtocols(args string) bool {
	for _, tok := range strings.Fields(args) {
		if tok == "TLSv1" || tok == "TLSv1.1" {
			return true
		}
	}
	return false
}

func parseSize(val string) *int64 {
	val = strings.ToLower(strings.TrimSpace(val))
	var multiplier float64 = 1
	switch {
	case strings.HasSuffix(val, "k"):
		multiplier = 1024
		val = val[:len(val)-1]
	case strings.HasSuffix(val, "m"):
		multiplier = 1024 * 1024
		val = val[:len(val)-1]
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil
	}
	result := int64(f * multiplier)
	return &result
}
