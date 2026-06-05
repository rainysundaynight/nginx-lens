package cli

import (
	"fmt"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newRouteCmd() *cobra.Command {
	var flagURL string
	cmd := &cobra.Command{
		Use:   "route [url]",
		Short: "Маршрутизация URL → server/location",
		Long:  "URL: аргумент, флаг --url или route.url в config.yaml.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			url := firstNonEmpty(argAt(args, 0), flagURL, cfg.Route.URL)
			if err := requireParam("url", url); err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			result := analyzer.FindRoute(tree, url)
			if result == nil {
				return fmt.Errorf("маршрут не найден для %s", url)
			}
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(result)
			case "yaml":
				return export.PrintYAML(result)
			}
			st := newStyler(cfg)
			printSection(st, "Route: "+url)
			loc := ""
			if result.Location != nil {
				loc = strings.TrimSpace(result.Location.Arg)
			}
			printKVTable(st, [][2]string{
				{"Server name", nodeDirective(result.Server, "server_name")},
				{"Listen", nodeDirective(result.Server, "listen")},
				{"Location", loc},
				{"Proxy pass", result.ProxyPass},
			})
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagURL, "url", "u", "", "URL для маршрутизации")
	return cmd
}
