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
		fmt.Println(st.ok("Проблем не найдено."))
		return
	}
	fmt.Printf("%s  %s  %s  %s\n", st.bold("TYPE"), st.bold("SEVERITY"), st.bold("FILE:LINE"), st.bold("DESCRIPTION"))
	fmt.Println(st.gray(stringsRepeat("─", 100)))
	for _, i := range issues {
		loc := ""
		if i.File != "" {
			loc = i.File
			if i.Line > 0 {
				loc += fmt.Sprintf(":%d", i.Line)
			}
		}
		fmt.Printf("%-28s %-8s %-12s %s\n", st.cyan(i.Type), st.severity(string(i.Severity)), st.gray(loc), i.Description)
	}
}

func stringsRepeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
