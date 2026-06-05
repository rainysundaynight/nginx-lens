package nginxinfo

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Группировка модулей и проверка директив ----------

// ModuleGroups — модули nginx по категориям.
type ModuleGroups struct {
	Core       []string `json:"core"`
	SSL        []string `json:"ssl"`
	Stream     []string `json:"stream"`
	ThirdParty []string `json:"third_party"`
}

// ModuleIssue — директива без собранного модуля.
type ModuleIssue struct {
	Directive string `json:"directive"`
	Module    string `json:"module"`
	Message   string `json:"message"`
}

var directiveModule = map[string]string{
	"proxy_pass":      "http_proxy",
	"grpc_pass":       "grpc",
	"fastcgi_pass":    "http_fastcgi",
	"uwsgi_pass":      "http_uwsgi",
	"scgi_pass":       "http_scgi",
	"memcached_pass":  "http_memcached",
	"ssl_certificate": "http_ssl",
	"limit_req":       "http_limit_req",
	"limit_conn":      "http_limit_conn",
	"real_ip_header":  "http_realip",
	"geo":             "http_geo",
	"stub_status":     "http_stub_status",
}

// GroupModules раскладывает модули по группам.
func GroupModules(build *BuildInfo) ModuleGroups {
	g := ModuleGroups{}
	if build == nil {
		return g
	}
	all := append(append([]string{}, build.StaticModules...), build.DynamicModules...)
	seen := make(map[string]struct{})
	for _, m := range all {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		cat := classifyModule(m)
		switch cat {
		case "ssl":
			g.SSL = append(g.SSL, m)
		case "stream":
			g.Stream = append(g.Stream, m)
		case "third":
			g.ThirdParty = append(g.ThirdParty, m)
		default:
			g.Core = append(g.Core, m)
		}
	}
	return g
}

func classifyModule(m string) string {
	ml := strings.ToLower(m)
	switch {
	case strings.Contains(ml, "ssl") || strings.Contains(ml, "tls"):
		return "ssl"
	case strings.HasPrefix(ml, "stream") || ml == "stream":
		return "stream"
	case strings.HasPrefix(ml, "http_") || strings.Contains(ml, "realip") || strings.Contains(ml, "stub_status"):
		return "core"
	default:
		if strings.Contains(ml, "brotli") || strings.Contains(ml, "geoip") || strings.Contains(ml, "njs") {
			return "third"
		}
		return "core"
	}
}

// HasModule проверяет наличие модуля в сборке.
func HasModule(build *BuildInfo, module string) bool {
	if build == nil {
		return false
	}
	module = strings.ToLower(module)
	for _, m := range build.StaticModules {
		if moduleMatches(m, module) {
			return true
		}
	}
	for _, m := range build.DynamicModules {
		if moduleMatches(m, module) {
			return true
		}
	}
	return false
}

func moduleMatches(have, need string) bool {
	h := strings.ToLower(strings.TrimSuffix(have, "_module"))
	n := strings.ToLower(strings.TrimSuffix(need, "_module"))
	if h == n {
		return true
	}
	if strings.HasSuffix(h, n) || strings.HasSuffix(n, h) {
		return true
	}
	return strings.Contains(h, n)
}

// CheckDirectiveModules находит директивы без модулей в сборке.
func CheckDirectiveModules(tree *parser.ConfigTree, build *BuildInfo) []ModuleIssue {
	if build == nil {
		return []ModuleIssue{{
			Module:  "nginx_build",
			Message: "nginx -V недоступен — проверка модулей пропущена",
		}}
	}
	var issues []ModuleIssue
	seen := make(map[string]struct{})
	for _, item := range analyzer.Walk(tree) {
		if item.Node.Block == "stream" {
			key := "block:stream"
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				if !HasModule(build, "stream") && !HasModule(build, "with-stream") {
					issues = append(issues, ModuleIssue{
						Directive: "stream",
						Module:    "stream",
						Message:   "блок stream используется, но модуль stream не найден в nginx -V",
					})
				}
			}
		}
		dir := item.Node.Directive
		if dir == "" {
			continue
		}
		mod, ok := directiveModule[dir]
		if !ok {
			continue
		}
		key := dir + "\x00" + mod
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if modulePresent(build, mod, dir) {
			continue
		}
		issues = append(issues, ModuleIssue{
			Directive: dir,
			Module:    mod,
			Message:   "директива " + dir + " используется, но модуль " + mod + " не найден в nginx -V",
		})
	}
	return issues
}

func modulePresent(build *BuildInfo, mod, dir string) bool {
	if HasModule(build, mod) || HasModule(build, dir) {
		return true
	}
	if mod == "grpc" {
		return HasModule(build, "http_v2")
	}
	return false
}
