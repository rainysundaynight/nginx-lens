package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newIncludeTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "include-tree",
		Short: "Дерево include-файлов (настройки: include_tree, defaults)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			path, err := nginxConfigPath(cfg)
			if err != nil {
				return err
			}
			tree := analyzer.BuildIncludeTree(path, nil)
			if cfg.IncludeTree.Directive != "" {
				shadows := analyzer.FindIncludeShadowing(tree, cfg.IncludeTree.Directive)
				switch outputFormat(cfg) {
				case "json":
					return export.PrintJSON(shadows)
				case "yaml":
					return export.PrintYAML(shadows)
				}
				st := newStyler(cfg)
				printSection(st, "Include shadowing: "+cfg.IncludeTree.Directive)
				if len(shadows) == 0 {
					printEmptyOK(st, "Перекрытий include не найдено.")
					return nil
				}
				rows := make([][]string, 0, len(shadows))
				for _, s := range shadows {
					rows = append(rows, []string{s.File, s.Directive, s.Value})
				}
				printTable(st, []int{32, 16, 0}, []string{"FILE", "DIRECTIVE", "VALUE"}, rows)
				return nil
			}
			cycles := analyzer.FindIncludeCycles(tree, nil)
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(map[string]interface{}{
					"tree":   tree,
					"cycles": cycles,
				})
			case "yaml":
				return export.PrintYAML(map[string]interface{}{
					"tree":   tree,
					"cycles": cycles,
				})
			}
			st := newStyler(cfg)
			printSection(st, "Include tree")
			if len(cycles) > 0 {
				printGroup(st, "Cycles")
				for _, c := range cycles {
					fmt.Printf("  %s %s\n", st.red("✗"), st.yellow(fmt.Sprint(c)))
				}
			}
			printIncludeTreeMap(st, tree, 0)
			return nil
		},
	}
}

func printIncludeTreeMap(st styler, tree analyzer.IncludeTree, depth int) {
	prefix := stringsRepeat("  ", depth)
	for path, v := range tree {
		switch val := v.(type) {
		case string:
			fmt.Printf("%s%s %s %s\n", prefix, st.cyan("include"), st.gray(path), st.yellow("→ "+val))
		case []analyzer.IncludeTree:
			fmt.Printf("%s%s %s\n", prefix, st.cyan("include"), st.gray(path))
			for _, sub := range val {
				printIncludeTreeMap(st, sub, depth+1)
			}
		default:
			fmt.Printf("%s%s %s\n", prefix, st.cyan("include"), st.gray(path))
		}
	}
}
