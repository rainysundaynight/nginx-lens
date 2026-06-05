package analyzer

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Маршрутизация URL ----------
// Определение server/location для заданного URL.

// RouteResult — результат маршрутизации URL.
type RouteResult struct {
	Server    *parser.Node `json:"server,omitempty"`
	Location  *parser.Node `json:"location,omitempty"`
	ProxyPass string       `json:"proxy_pass,omitempty"`
}

// FindRoute находит server и location для URL через route engine.
func FindRoute(tree *parser.ConfigTree, rawURL string) *RouteResult {
	ex := ExplainRoute(tree, rawURL)
	if ex == nil || ex.Server == nil {
		return nil
	}
	return &RouteResult{
		Server:    ex.Server,
		Location:  ex.Location,
		ProxyPass: ex.ProxyPass,
	}
}

func hostMatch(host, pattern string) bool {
	if pattern == "_" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		return strings.HasSuffix(host, pattern[1:])
	}
	return host == pattern
}
