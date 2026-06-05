package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"github.com/spf13/cobra"
)

func newResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve",
		Short: "DNS-резолвинг upstream (настройки: defaults, resolve, cache, output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			setupDNSCache(cfg, cfg.Resolve.SkipCache)

			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			results := upstream.ResolveUpstreams(
				tree.GetUpstreams(),
				cfg.Defaults.MaxWorkers,
				cfg.Cache.Enabled && !cfg.Resolve.SkipCache,
				cfg.Cache.TTL,
				"",
			)

			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(export.ResolveExport{Upstreams: results})
			case "yaml":
				return export.PrintYAML(export.ResolveExport{Upstreams: results})
			}
			for name, servers := range results {
				fmt.Printf("\n[%s]\n", name)
				for _, s := range servers {
					fmt.Printf("  %s → %v\n", s.Address, s.Resolved)
				}
			}
			return nil
		},
	}
}
