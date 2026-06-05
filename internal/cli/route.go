package cli

import (
	"encoding/json"
	"fmt"

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
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagURL, "url", "u", "", "URL для маршрутизации")
	return cmd
}
