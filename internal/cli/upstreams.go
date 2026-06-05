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
			st := newStyler(cfg)
			printSection(st, "Upstreams")
			for _, b := range output {
				printGroup(st, b.Name+" ("+b.File+")")
				if len(b.Servers) > 0 {
					healthByAddr := map[string]upstream.ServerHealth{}
					if runHealth {
						for _, hs := range b.Health[b.Name] {
							healthByAddr[hs.Address] = hs
						}
					}
					rows := make([][]string, 0, len(b.Servers))
					for _, s := range b.Servers {
						status := st.gray("—")
						if hs, ok := healthByAddr[s]; ok {
							status = statusLabel(st, hs.Healthy)
						}
						rows = append(rows, []string{s, status})
					}
					printTable(st, []int{28, 8}, []string{"SERVER", "STATUS"}, rows)
				}
				if len(b.Refs) > 0 {
					refRows := make([][]string, 0, len(b.Refs))
					for _, r := range b.Refs {
						refRows = append(refRows, []string{r.FromDirective, r.Value})
					}
					printSummary(st, "References:")
					printTable(st, []int{16, 0}, []string{"FROM", "TARGET"}, refRows)
				}
			}
			return nil
		},
	}
}
