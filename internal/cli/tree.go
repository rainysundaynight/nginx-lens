package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree",
		Short: "Визуализация дерева конфигурации (настройки: tree.format, defaults)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			switch cfg.Tree.Format {
			case "markdown":
				fmt.Print(export.RenderTreeMarkdown(tree.Directives, 0))
			case "html":
				page := "<!DOCTYPE html><html><body>" + export.RenderTreeHTML(tree.Directives) + "</body></html>"
				fmt.Print(page)
			default:
				fmt.Print(export.RenderTreeText(tree.Directives, "", true))
			}
			return nil
		},
	}
}
