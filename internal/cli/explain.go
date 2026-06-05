package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newExplainCmd() *cobra.Command {
	var flagURL string
	cmd := &cobra.Command{
		Use:   "explain [url]",
		Short: "Пошаговое объяснение маршрутизации URL",
		Long:  "URL: аргумент, флаг --url, explain.url или route.url в config.yaml.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			url := firstNonEmpty(argAt(args, 0), flagURL, cfg.Explain.URL, cfg.Route.URL)
			if err := requireParam("url", url); err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			result := analyzer.ExplainRoute(tree, url)
			if result == nil {
				return fmt.Errorf("не удалось разобрать URL: %s", url)
			}
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(result)
			case "yaml":
				return export.PrintYAML(result)
			}
			st := newStyler(cfg)
			fmt.Println(st.header("Trace: " + url))
			for _, step := range result.Trace {
				mark := st.gray("·")
				if step.Matched {
					mark = st.green("✓")
				}
				fmt.Printf("%s %s %s\n", mark, st.cyan("["+step.Step+"]"), step.Detail)
			}
			if result.Location != nil {
				fmt.Printf("\n%s location %s%s\n", st.bold("→"), result.Location.LocModifier, result.Location.Arg)
			}
			if result.Upstream != "" {
				fmt.Printf("%s upstream: %s\n", st.bold("→"), st.green(result.Upstream))
			} else if result.ProxyPass != "" {
				fmt.Printf("%s proxy_pass: %s\n", st.bold("→"), st.green(result.ProxyPass))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagURL, "url", "u", "", "URL для trace маршрутизации")
	return cmd
}
