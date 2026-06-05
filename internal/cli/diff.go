package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/nginxload"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var flagConfig1, flagConfig2 string
	cmd := &cobra.Command{
		Use:   "diff [config1] [config2]",
		Short: "Семантическое сравнение конфигов (upstream/location/server)",
		Long:  "Сравнивает два nginx.conf. Пути: аргументы, флаги --config1/--config2 или diff.config1/config2 в config.yaml.",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			config1 := firstNonEmpty(argAt(args, 0), flagConfig1, cfg.Diff.Config1)
			config2 := firstNonEmpty(argAt(args, 1), flagConfig2, cfg.Diff.Config2)
			if err := requireParam("config1", config1); err != nil {
				return err
			}
			if err := requireParam("config2", config2); err != nil {
				return err
			}

			tree1, err1 := nginxload.ParseDiffFile(cfg, config1)
			tree2, err2 := nginxload.ParseDiffFile(cfg, config2)
			if err1 != nil {
				return err1
			}
			if err2 != nil {
				return err2
			}

			diffs := analyzer.DiffSemantic(tree1, tree2)
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(diffs)
			case "yaml":
				return export.PrintYAML(diffs)
			}

			st := newStyler(cfg)
			if len(diffs) == 0 {
				printEmptyOK(st, "Структурных отличий не найдено")
				return nil
			}
			printSection(st, fmt.Sprintf("Diff (%d changes)", len(diffs)))
			for _, d := range diffs {
				printDiffLine(st, d)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagConfig1, "config1", "", "Первый конфиг (nginx.conf)")
	cmd.Flags().StringVar(&flagConfig2, "config2", "", "Второй конфиг для сравнения")
	return cmd
}

func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func printDiffLine(st styler, d analyzer.TreeDiff) {
	path := st.pathParts(d.Path)
	switch d.Type {
	case "added":
		fmt.Printf("%s  %s\n", st.diffType(d.Type), st.green(path))
		if v, ok := d.Value2.(string); ok && v != "" {
			fmt.Printf("    %s\n", st.green(v))
		}
	case "removed":
		fmt.Printf("%s  %s\n", st.diffType(d.Type), st.red(path))
		if v, ok := d.Value1.(string); ok && v != "" {
			fmt.Printf("    %s\n", st.red(v))
		}
	case "changed":
		fmt.Printf("%s  %s\n", st.diffType(d.Type), st.yellow(path))
		if v1, ok := d.Value1.(string); ok && v1 != "" {
			fmt.Printf("    %s %s\n", st.red("-"), st.red(v1))
		}
		if v2, ok := d.Value2.(string); ok && v2 != "" {
			fmt.Printf("    %s %s\n", st.green("+"), st.green(v2))
		}
	default:
		fmt.Printf("[%s] %s\n", d.Type, path)
	}
}
