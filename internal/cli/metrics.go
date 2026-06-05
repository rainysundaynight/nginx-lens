package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/nginxload"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/spf13/cobra"
)

func newMetricsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "metrics",
		Short: "Метрики конфигурации (настройки: metrics, defaults, output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			metrics := computeMetrics(tree)
			if cfg.Metrics.ComparePath != "" {
				if tree2, err := nginxload.ParseFile(cfg, cfg.Metrics.ComparePath); err == nil {
					metrics["compare"] = computeMetrics(tree2)
				}
			}
			if cfg.Metrics.Prometheus || outputFormat(cfg) == "prometheus" {
				prom := map[string]float64{
					"nginx_lens_servers_total":    float64(metrics["servers"].(int)),
					"nginx_lens_locations_total":  float64(metrics["locations"].(int)),
					"nginx_lens_upstreams_total":  float64(metrics["upstreams"].(int)),
					"nginx_lens_directives_total": float64(metrics["directives"].(int)),
				}
				fmt.Print(export.FormatPrometheus(prom, nil))
				return nil
			}
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(metrics)
			case "yaml":
				return export.PrintYAML(metrics)
			}
			st := newStyler(cfg)
			printSection(st, "Configuration metrics")
			keys := []string{"servers", "locations", "upstreams", "directives"}
			rows := make([][]string, 0, len(keys))
			for _, k := range keys {
				if v, ok := metrics[k]; ok {
					rows = append(rows, []string{k, fmt.Sprintf("%v", v)})
				}
			}
			printTable(st, []int{16, 12}, []string{"METRIC", "VALUE"}, rows)
			return nil
		},
	}
}

func computeMetrics(tree *parser.ConfigTree) map[string]interface{} {
	servers, locations, directives := 0, 0, 0
	dirCounts := make(map[string]int)
	for _, item := range analyzer.Walk(tree) {
		switch item.Node.Block {
		case "server":
			servers++
		case "location":
			locations++
		}
		if item.Node.Directive != "" {
			directives++
			dirCounts[item.Node.Directive]++
		}
	}
	return map[string]interface{}{
		"servers":          servers,
		"locations":        locations,
		"upstreams":        len(tree.GetUpstreams()),
		"directives":       directives,
		"directive_counts": dirCounts,
	}
}
