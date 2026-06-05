package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"github.com/spf13/cobra"
)

func newBlastRadiusCmd() *cobra.Command {
	var flagUpstream string
	cmd := &cobra.Command{
		Use:   "blast-radius [upstream]",
		Short: "Blast-radius падения upstream",
		Long:  "Имя upstream: аргумент, флаг --upstream или blast_radius.upstream_name в config.yaml.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			name := firstNonEmpty(argAt(args, 0), flagUpstream, cfg.BlastRadius.UpstreamName)
			if err := requireParam("upstream", name); err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			ups := tree.GetUpstreams()
			health := upstream.CheckUpstreams(
				ups,
				cfg.Defaults.Timeout,
				cfg.Defaults.Retries,
				cfg.Defaults.Mode,
				cfg.Defaults.MaxWorkers,
			)
			report := analyzer.BuildBlastRadius(tree, name, health)
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(report)
			case "yaml":
				return export.PrintYAML(report)
			}
			st := newStyler(cfg)
			status := statusLabel(st, report.Healthy)
			printSection(st, "Blast-radius: "+report.Upstream)
			printKVTable(st, [][2]string{
				{"Upstream", st.cyan(report.Upstream)},
				{"Status", status},
				{"Impacted locations", fmt.Sprintf("%d", len(report.Impact))},
			})
			if len(report.Impact) > 0 {
				rows := make([][]string, 0, len(report.Impact))
				for _, e := range report.Impact {
					rows = append(rows, []string{
						e.ServerName + ":" + e.Listen,
						e.Location,
						e.ProxyPass,
					})
				}
				fmt.Println()
				printTable(st, []int{24, 20, 0}, []string{"SERVER", "LOCATION", "PROXY_PASS"}, rows)
			} else {
				printSummary(st, "Нет ссылок на этот upstream")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagUpstream, "upstream", "U", "", "Имя upstream-блока")
	return cmd
}
