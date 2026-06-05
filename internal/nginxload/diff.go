package nginxload

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ParseDiffFile парсит файл для diff: всегда с диска (regex), без docker/nginx -T.
func ParseDiffFile(cfg config.Config, path string) (*parser.ConfigTree, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("diff: файл не найден: %s", path)
	}
	opts := parser.ParseOptions{Mode: "regex", NginxPath: cfg.Parser.NginxPath}
	tree, err := parser.ParseConfig(abs, opts)
	if err != nil {
		return nil, err
	}
	dyn := cfg.DynamicUpstream
	tree.SetDynamicUpstreamConfig(dyn.Enabled, dyn.APIURL, dyn.Timeout)
	return tree, nil
}
