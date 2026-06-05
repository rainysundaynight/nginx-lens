package cli

import (
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
			st := newStyler(cfg)
			printSection(st, "Route graph")
			for _, item := range analyzer.Walk(tree) {
				if item.Node.Block != "server" {
					continue
				}
				serverLabel := item.Node.Arg
				if serverLabel == "" {
					serverLabel = "(default)"
				}
				printGroup(st, "server "+serverLabel)
				rows := make([][]string, 0)
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
					rows = append(rows, []string{sub.Node.Arg, proxyPass})
				}
				if len(rows) > 0 {
					printTable(st, []int{24, 0}, []string{"LOCATION", "PROXY_PASS"}, rows)
				}
			}
			return nil
		},
	}
}
