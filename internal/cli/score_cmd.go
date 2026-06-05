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
			printSection(st, "Config score")
			scoreColor := st.green
			if report.Total < 70 {
				scoreColor = st.yellow
			}
			if report.Total < 50 {
				scoreColor = st.red
			}
			printKVTable(st, [][2]string{
				{"Total", scoreColor(fmt.Sprintf("%.0f/100", report.Total))},
			})
			rows := make([][]string, 0, len(report.Categories))
			for _, c := range report.Categories {
				rows = append(rows, []string{
					st.cyan(c.Name),
					st.bold(fmt.Sprintf("%.0f", c.Score)),
					fmt.Sprintf("%.0f%%", c.Weight*100),
					fmt.Sprintf("%d", c.Issues),
				})
			}
			fmt.Println()
			printTable(st, []int{16, 8, 10, 8}, []string{"CATEGORY", "SCORE", "WEIGHT", "ISSUES"}, rows)
			if len(report.TopActions) > 0 {
				printSection(st, "Top recommendations")
				for i, a := range report.TopActions {
					fmt.Printf("  %d. %s\n", i+1, st.yellow(a))
				}
			}
			return nil
		},
	}
}
