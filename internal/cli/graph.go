package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "graph",
		Short: "Граф маршрутов (настройки: defaults.nginx_config_path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			for _, item := range analyzer.Walk(tree) {
				if item.Node.Block != "server" {
					continue
				}
				serverLabel := item.Node.Arg
				if serverLabel == "" {
					serverLabel = "(default)"
				}
				fmt.Printf("server %s\n", serverLabel)
				for _, sub := range analyzer.WalkNodes(item.Node.Directives, &item.Node) {
					if sub.Node.Block != "location" {
						continue
					}
					proxyPass := ""
					for _, inner := range analyzer.WalkNodes(sub.Node.Directives, &sub.Node) {
						if inner.Node.Directive == "proxy_pass" {
							proxyPass = inner.Node.Args
							break
						}
					}
					fmt.Printf("  location %s → %s\n", sub.Node.Arg, proxyPass)
				}
			}
			return nil
		},
	}
}
