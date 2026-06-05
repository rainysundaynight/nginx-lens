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
			status := st.green("UP")
			if !report.Healthy {
				status = st.red("DOWN")
			}
			fmt.Printf("upstream %s (%s)\n", st.cyan(report.Upstream), status)
			for _, e := range report.Impact {
				fmt.Printf("  %s server %s:%s\n", st.gray("└──"), e.ServerName, e.Listen)
				fmt.Printf("        %s location %s → %s\n", st.gray("└──"), st.yellow(e.Location), st.green(e.ProxyPass))
			}
			if len(report.Impact) == 0 {
				fmt.Println(st.gray("  (нет ссылок на этот upstream)"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagUpstream, "upstream", "U", "", "Имя upstream-блока")
	return cmd
}
