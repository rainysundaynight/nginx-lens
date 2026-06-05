package nginxload

import (
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Единый парсинг произвольного пути ----------
// Используется diff/metrics вместо прямого ParseNginxConfig.

// ParseFile парсит nginx.conf по пути с учётом parser.mode.
func ParseFile(cfg config.Config, path string) (*parser.ConfigTree, error) {
	opts := parser.ParseOptions{Mode: cfg.Parser.Mode, NginxPath: cfg.Parser.NginxPath}
	tree, err := parser.ParseConfig(path, opts)
	if err != nil {
		return nil, err
	}
	dyn := cfg.DynamicUpstream
	tree.SetDynamicUpstreamConfig(dyn.Enabled, dyn.APIURL, dyn.Timeout)
	return tree, nil
}
