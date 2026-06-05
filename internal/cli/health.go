package cli

import (
	"os"

	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Проверка доступности upstream (настройки: defaults, health, cache, output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			setupDNSCache(cfg, cfg.Health.SkipCache)

			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			ups := tree.GetUpstreams()
			results := upstream.CheckUpstreams(
				ups,
				cfg.Defaults.Timeout,
				cfg.Defaults.Retries,
				cfg.Defaults.Mode,
				cfg.Defaults.MaxWorkers,
			)

			if cfg.Health.WithResolve {
				resolved := upstream.ResolveUpstreams(
					ups,
					cfg.Defaults.MaxWorkers,
					cfg.Cache.Enabled && !cfg.Health.SkipCache,
					cfg.Cache.TTL,
					"",
				)
				switch outputFormat(cfg) {
				case "json":
					return export.PrintJSON(map[string]interface{}{
						"health": results, "resolve": resolved,
					})
				case "yaml":
					return export.PrintYAML(map[string]interface{}{
						"health": results, "resolve": resolved,
					})
				}
			}

			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(export.HealthExport{Upstreams: results})
			case "yaml":
				return export.PrintYAML(export.HealthExport{Upstreams: results})
			}

			unhealthy := printHealthTable(cfg, results)
			if unhealthy && !cfg.Health.SkipExitOnUnhealthy {
				os.Exit(1)
			}
			return nil
		},
	}
}

func printHealthTable(cfg config.Config, results map[string][]upstream.ServerHealth) bool {
	st := newStyler(cfg)
	printSection(st, "Upstream health")
	unhealthy := false
	for name, servers := range results {
		printGroup(st, name)
		rows := make([][]string, 0, len(servers))
		for _, s := range servers {
			if !s.Healthy {
				unhealthy = true
			}
			rows = append(rows, []string{s.Address, statusLabel(st, s.Healthy)})
		}
		printTable(st, []int{28, 8}, []string{"ADDRESS", "STATUS"}, rows)
	}
	return unhealthy
}
