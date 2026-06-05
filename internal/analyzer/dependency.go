package analyzer

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

// ---------- Blast-radius граф ----------
// Зависимости upstream → location → server.

// BlastRadiusEntry — затронутый путь при падении upstream.
type BlastRadiusEntry struct {
	UpstreamName string `json:"upstream_name"`
	ServerName   string `json:"server_name"`
	Listen       string `json:"listen"`
	Location     string `json:"location"`
	FromDirective string `json:"from_directive"`
	ProxyPass    string `json:"proxy_pass"`
	ConfigFile   string `json:"config_file"`
	Healthy      *bool  `json:"healthy,omitempty"`
}

// BlastRadiusReport — отчёт blast-radius.
type BlastRadiusReport struct {
	Upstream string             `json:"upstream"`
	Healthy  bool               `json:"healthy"`
	Impact   []BlastRadiusEntry `json:"impact"`
}

// BuildBlastRadius строит blast-radius для upstream.
func BuildBlastRadius(tree *parser.ConfigTree, upstreamName string, health map[string][]upstream.ServerHealth) BlastRadiusReport {
	names := upstream.CollectKnownUpstreamNames(tree)
	refs := upstream.FindUpstreamReferences(tree, names)
	report := BlastRadiusReport{Upstream: upstreamName, Healthy: true}

	if health != nil {
		for _, rows := range health[upstreamName] {
			if !rows.Healthy {
				report.Healthy = false
				break
			}
		}
	}

	for _, ref := range refs {
		if ref.UpstreamName != upstreamName {
			continue
		}
		entry := BlastRadiusEntry{
			UpstreamName:  ref.UpstreamName,
			ServerName:    ref.ServerName,
			Listen:        ref.Listen,
			Location:      ref.Location,
			FromDirective: ref.FromDirective,
			ProxyPass:     ref.Value,
			ConfigFile:    ref.ConfigFile,
		}
		report.Impact = append(report.Impact, entry)
	}
	return report
}

// BuildDependencyGraph строит полный граф зависимостей.
func BuildDependencyGraph(tree *parser.ConfigTree) map[string][]BlastRadiusEntry {
	names := upstream.CollectKnownUpstreamNames(tree)
	refs := upstream.FindUpstreamReferences(tree, names)
	graph := make(map[string][]BlastRadiusEntry)
	for _, ref := range refs {
		graph[ref.UpstreamName] = append(graph[ref.UpstreamName], BlastRadiusEntry{
			UpstreamName:  ref.UpstreamName,
			ServerName:    ref.ServerName,
			Listen:        ref.Listen,
			Location:      ref.Location,
			FromDirective: ref.FromDirective,
			ProxyPass:     ref.Value,
			ConfigFile:    ref.ConfigFile,
		})
	}
	return graph
}

// FindUpstreamFromProxyPass извлекает имя upstream из proxy_pass.
func FindUpstreamFromProxyPass(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		rest := strings.SplitN(value, "://", 2)[1]
		if idx := strings.Index(rest, "/"); idx >= 0 {
			rest = rest[:idx]
		}
		return rest
	}
	return strings.Fields(value)[0]
}
