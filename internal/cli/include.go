package cli

import (
	"encoding/json"
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
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
				data, _ := json.MarshalIndent(shadows, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			cycles := analyzer.FindIncludeCycles(tree, nil)
			if len(cycles) > 0 {
				fmt.Println("Циклы include:")
				for _, c := range cycles {
					fmt.Println(" ", c)
				}
			}
			data, _ := json.MarshalIndent(tree, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
}
