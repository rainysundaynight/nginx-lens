package cli

import (
	"strings"

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
			st := newStyler(cfg)
			printSection(st, "DNS resolve")
			for name, servers := range results {
				printGroup(st, name)
				rows := make([][]string, 0, len(servers))
				for _, s := range servers {
					rows = append(rows, []string{s.Address, strings.Join(s.Resolved, ", ")})
				}
				printTable(st, []int{28, 0}, []string{"ADDRESS", "RESOLVED"}, rows)
			}
			return nil
		},
	}
}
