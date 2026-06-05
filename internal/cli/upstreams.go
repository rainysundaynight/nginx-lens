package cli

import (
	"encoding/json"
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"github.com/spf13/cobra"
)

func newUpstreamsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upstreams",
		Short: "Сводка по upstream-блокам (настройки: upstreams, defaults, output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			blocks := upstream.FindUpstreamBlocks(tree)
			names := upstream.CollectKnownUpstreamNames(tree)
			refs := upstream.FindUpstreamReferences(tree, names)

			if cfg.Upstreams.Name != "" {
				var filtered []upstream.UpstreamBlock
				for _, b := range blocks {
					if b.Name == cfg.Upstreams.Name {
						filtered = append(filtered, b)
					}
				}
				blocks = filtered
			}

			runHealth := cfg.Upstreams.Health || (cfg.Upstreams.HealthByDefault && !cfg.Upstreams.SkipHealth)

			type blockOut struct {
				upstream.UpstreamBlock
				Health map[string][]upstream.ServerHealth `json:"health,omitempty"`
			}
			var output []blockOut
			for _, b := range blocks {
				out := blockOut{UpstreamBlock: b}
				for _, r := range refs {
					if r.UpstreamName == b.Name {
						b.Refs = append(b.Refs, r)
					}
				}
				out.UpstreamBlock = b
				if runHealth {
					ups := map[string][]string{b.Name: b.Servers}
					out.Health = upstream.CheckUpstreams(
						ups,
						cfg.Defaults.Timeout,
						cfg.Defaults.Retries,
						cfg.Defaults.Mode,
						cfg.Defaults.MaxWorkers,
					)
				}
				output = append(output, out)
			}

			switch outputFormat(cfg) {
			case "json":
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			case "yaml":
				return export.PrintYAML(output)
			}
			for _, b := range output {
				fmt.Printf("\nupstream %s (%s)\n", b.Name, b.File)
				for _, s := range b.Servers {
					fmt.Printf("  server %s\n", s)
				}
				if len(b.Refs) > 0 {
					fmt.Println("  references:")
					for _, r := range b.Refs {
						fmt.Printf("    %s → %s\n", r.FromDirective, r.Value)
					}
				}
			}
			return nil
		},
	}
}
