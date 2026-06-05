package policy

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Policy engine ----------
// YAML-правила и встроенные packs.

// Issue — нарушение политики.
type Issue struct {
	RuleID      string            `json:"rule_id"`
	Type        string            `json:"type"`
	Severity    analyzer.Severity `json:"severity"`
	Message     string            `json:"message"`
	FixHint     string            `json:"fix_hint,omitempty"`
	File        string            `json:"file,omitempty"`
	Line        int               `json:"line,omitempty"`
	Directive   string            `json:"directive,omitempty"`
}

// Rule — правило политики.
type Rule struct {
	ID       string
	Match    string
	Severity analyzer.Severity
	Message  string
	FixHint  string
	Check    func(item analyzer.WalkItem) bool
}

// Engine выполняет policy checks.
type Engine struct {
	rules []Rule
}

// NewEngine создаёт engine с packs и custom rules.
func NewEngine(packs []string, custom []CustomRule) *Engine {
	var rules []Rule
	for _, pack := range packs {
		rules = append(rules, builtinPack(pack)...)
	}
	for _, c := range custom {
		rules = append(rules, compileCustomRule(c))
	}
	return &Engine{rules: rules}
}

// CustomRule — пользовательское правило из config.
type CustomRule struct {
	ID       string
	Match    string
	Severity string
	Message  string
	FixHint  string
}

// Run выполняет все правила над деревом конфигурации.
func (e *Engine) Run(tree *parser.ConfigTree) []Issue {
	var issues []Issue
	for _, item := range analyzer.Walk(tree) {
		for _, rule := range e.rules {
			if rule.Check != nil && rule.Check(item) {
				issues = append(issues, Issue{
					RuleID:    rule.ID,
					Type:      rule.ID,
					Severity:  rule.Severity,
					Message:   rule.Message,
					FixHint:   rule.FixHint,
					File:      item.Node.File,
					Line:      item.Node.Line,
					Directive: item.Node.Directive,
				})
			} else if rule.Match != "" && matchSimple(rule.Match, item) {
				issues = append(issues, Issue{
					RuleID:    rule.ID,
					Type:      rule.ID,
					Severity:  rule.Severity,
					Message:   rule.Message,
					FixHint:   rule.FixHint,
					File:      item.Node.File,
					Line:      item.Node.Line,
					Directive: item.Node.Directive,
				})
			}
		}
	}
	return issues
}

func matchSimple(match string, item analyzer.WalkItem) bool {
	if strings.HasPrefix(match, "directive.") && strings.Contains(match, " contains ") {
		idx := strings.Index(match, " contains ")
		dir := strings.TrimPrefix(match[:idx], "directive.")
		needle := strings.TrimSpace(match[idx+len(" contains "):])
		return item.Node.Directive == dir && strings.Contains(item.Node.Args, needle)
	}
	if strings.HasPrefix(match, "directive.") {
		dir := strings.TrimPrefix(match, "directive.")
		return item.Node.Directive == dir
	}
	return false
}

func compileCustomRule(c CustomRule) Rule {
	sev := analyzer.SeverityMedium
	switch c.Severity {
	case "high":
		sev = analyzer.SeverityHigh
	case "low":
		sev = analyzer.SeverityLow
	}
	return Rule{ID: c.ID, Match: c.Match, Severity: sev, Message: c.Message, FixHint: c.FixHint}
}

func builtinPack(name string) []Rule {
	switch name {
	case "security-baseline":
		return []Rule{
			{ID: "no_autoindex", Severity: analyzer.SeverityMedium, Message: "autoindex on запрещён",
				FixHint: "autoindex off;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "autoindex" && strings.TrimSpace(item.Node.Args) == "on"
				}},
			{ID: "no_server_tokens", Severity: analyzer.SeverityLow, Message: "server_tokens on раскрывает версию",
				FixHint: "server_tokens off;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "server_tokens" && strings.TrimSpace(item.Node.Args) == "on"
				}},
		}
	case "mozilla-ssl":
		return []Rule{
			{ID: "no_tls10", Severity: analyzer.SeverityHigh, Message: "TLS 1.0/1.1 запрещён Mozilla guidelines",
				FixHint: "ssl_protocols TLSv1.2 TLSv1.3;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "ssl_protocols" &&
						(strings.Contains(item.Node.Args, "TLSv1") && !strings.Contains(item.Node.Args, "TLSv1.2"))
				}},
		}
	case "performance", "performance-baseline":
		return []Rule{
			{ID: "upstream_keepalive", Severity: analyzer.SeverityLow, Message: "upstream без keepalive",
				FixHint: "keepalive 32;",
				Check: func(item analyzer.WalkItem) bool {
					if item.Node.Upstream == "" {
						return false
					}
					for _, o := range item.Node.Options {
						if o.Name == "keepalive" {
							return false
						}
					}
					return true
				}},
			{ID: "worker_connections", Severity: analyzer.SeverityLow, Message: "низкий worker_connections",
				FixHint: "events { worker_connections 4096; }",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "worker_connections" && strings.TrimSpace(item.Node.Args) == "512"
				}},
		}
	case "caching":
		return []Rule{
			{ID: "proxy_cache_path_missing", Severity: analyzer.SeverityLow, Message: "proxy_cache без proxy_cache_path",
				FixHint: "proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=cache:10m;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "proxy_cache" && strings.TrimSpace(item.Node.Args) != "off"
				}},
			{ID: "expires_missing", Severity: analyzer.SeverityLow, Message: "статика без expires/cache",
				FixHint: "expires 7d; или proxy_cache_valid 200 1h;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Block == "location" && strings.Contains(item.Node.Arg, "/static")
				}},
		}
	case "rate-limit":
		return []Rule{
			{ID: "limit_req_zone_missing", Severity: analyzer.SeverityMedium, Message: "limit_req без limit_req_zone",
				FixHint: "limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "limit_req"
				}},
			{ID: "limit_conn_zone_missing", Severity: analyzer.SeverityLow, Message: "limit_conn без limit_conn_zone",
				FixHint: "limit_conn_zone $binary_remote_addr zone=conn:10m;",
				Check: func(item analyzer.WalkItem) bool {
					return item.Node.Directive == "limit_conn"
				}},
		}
	}
	return nil
}
