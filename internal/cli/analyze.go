package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Статический анализ конфигурации (настройки: defaults, analyze, output)",
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
			exportData := export.FormatAnalyzeResults(result, analyzeFilter(cfg))

			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(exportData)
			case "yaml":
				return export.PrintYAML(exportData)
			}
			printAnalyzeTable(cfg, exportData.Issues)
			return nil
		},
	}
}

func printAnalyzeTable(cfg config.Config, issues []analyzer.Issue) {
	st := newStyler(cfg)
	if len(issues) == 0 {
		printEmptyOK(st, "Проблем не найдено.")
		return
	}
	printSection(st, "Analysis issues")
	rows := make([][]string, 0, len(issues))
	for _, i := range issues {
		loc := issueLocation(i)
		rows = append(rows, []string{st.cyan(i.Type), st.severity(string(i.Severity)), st.gray(loc), i.Description})
	}
	printTable(st, []int{28, 8, 14, 0}, []string{"TYPE", "SEVERITY", "FILE:LINE", "DESCRIPTION"}, rows)
}

func issueLocation(i analyzer.Issue) string {
	if i.File == "" {
		return ""
	}
	if i.Line > 0 {
		return fmt.Sprintf("%s:%d", i.File, i.Line)
	}
	return i.File
}
