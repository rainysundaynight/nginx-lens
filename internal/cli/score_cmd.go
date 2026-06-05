package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newScoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "score",
		Short: "Рейтинг готовности конфигурации 0-100 (настройки: score)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			result := analyzer.RunAnalysis(tree)
			certIssues := analyzer.AuditCertificates(tree, cfg.Certs.WarnDays, certVolumeMap(cfg))
			certHigh := 0
			for _, c := range certIssues {
				if c.Severity == analyzer.SeverityHigh {
					certHigh++
				}
			}
			engine := policyEngineFromCfg(cfg)
			policyIssues := engine.Run(tree)
			report := analyzer.ComputeScore(result, len(policyIssues), certHigh)
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(report)
			case "yaml":
				return export.PrintYAML(report)
			}
			st := newStyler(cfg)
			scoreColor := st.green
			if report.Total < 70 {
				scoreColor = st.yellow
			}
			if report.Total < 50 {
				scoreColor = st.red
			}
			fmt.Printf("Config Score: %s\n\n", scoreColor(fmt.Sprintf("%.0f/100", report.Total)))
			for _, c := range report.Categories {
				fmt.Printf("  %-16s %s (weight %.0f%%, issues %d)\n", st.cyan(c.Name), st.bold(fmt.Sprintf("%.0f", c.Score)), c.Weight*100, c.Issues)
			}
			if len(report.TopActions) > 0 {
				fmt.Println("\n" + st.header("Top recommendations:"))
				for i, a := range report.TopActions {
					fmt.Printf("  %d. %s\n", i+1, st.yellow(a))
				}
			}
			return nil
		},
	}
}
