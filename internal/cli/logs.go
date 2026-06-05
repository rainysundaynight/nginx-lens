package cli

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/logs"
	"github.com/rainysundaynight/nginx-lens/internal/nginxload"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Статистика access/error логов (настройки: logs, output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			if cfg.Logs.Path == "" && cfg.Logs.ErrorPath == "" {
				return requireNonEmpty("logs.path", "")
			}
			tailLines := cfg.Logs.TailLines
			if tailLines <= 0 {
				tailLines = 50000
			}
			top := cfg.Logs.Top
			if top == 0 {
				top = cfg.Defaults.Top
			}

			report := map[string]interface{}{}
			if cfg.Logs.Path != "" {
				raw, err := nginxload.ReadLogTail(cfg, cfg.Logs.Path, tailLines)
				if err != nil {
					return err
				}
				accessSnap, err := logs.ParseAccessSnapshotContent(raw, cfg.Logs.Since, cfg.Logs.Until, cfg.Logs.Status, cfg.Logs.FormatRegex)
				if err != nil {
					return err
				}
				report["access_snapshot"] = accessSnap
				stats, err := logs.ParseLogFile(
					nginxload.ResolveLogPath(cfg, cfg.Logs.Path),
					top,
					cfg.Logs.Since,
					cfg.Logs.Until,
					cfg.Logs.Status,
					cfg.Logs.SkipAnomalies,
				)
				if err == nil {
					report["access"] = stats
				}
			}
			if cfg.Logs.ErrorPath != "" {
				raw, err := nginxload.ReadLogTail(cfg, cfg.Logs.ErrorPath, tailLines)
				if err != nil {
					return err
				}
				errorStats, err := logs.ParseErrorLogContent(raw)
				if err != nil {
					return err
				}
				report["error"] = errorStats
			}
			tree, treeErr := parseConfigFromCfg(cfg)
			if treeErr == nil {
				ups := tree.GetUpstreams()
				depGraph := analyzer.BuildDependencyGraph(tree)
				streamGraph := upstream.BuildStreamGraph(tree)
				if snap, ok := report["access_snapshot"].(*logs.AccessSnapshot); ok {
					errStats, _ := report["error"].(*logs.ErrorStats)
					report["correlations"] = logs.BuildCorrelations(snap, errStats, depGraph, ups, streamGraph)
				} else if errStats, ok := report["error"].(*logs.ErrorStats); ok {
					report["correlations"] = logs.BuildCorrelations(nil, errStats, depGraph, ups, streamGraph)
				}
			}

			switch outputFormat(cfg) {
			case "csv":
				if stats, ok := report["access"].(*logs.Stats); ok {
					var rows [][]string
					for _, e := range stats.TopPaths {
						rows = append(rows, []string{e.Key, fmt.Sprintf("%d", e.Count)})
					}
					return export.WriteCSV([]string{"path", "count"}, rows)
				}
				return export.PrintJSON(report)
			case "json":
				return export.PrintJSON(report)
			case "yaml":
				return export.PrintYAML(report)
			}

			st := newStyler(cfg)
			printSection(st, "Access & error logs")
			if snap, ok := report["access_snapshot"].(*logs.AccessSnapshot); ok {
				printKVTable(st, [][2]string{
					{"Access requests", fmt.Sprintf("%d", snap.TotalRequests)},
					{"RPS", fmt.Sprintf("%.2f", snap.RPS)},
					{"404", fmt.Sprintf("%d", snap.Status404)},
					{"5xx", fmt.Sprintf("%d", snap.Status5xx)},
					{"P95 latency", fmt.Sprintf("%.1f ms", snap.P95Latency)},
				})
			}
			if errStats, ok := report["error"].(*logs.ErrorStats); ok {
				fmt.Println()
				printGroup(st, "Error log")
				printKVTable(st, [][2]string{
					{"Total errors", fmt.Sprintf("%d", errStats.TotalErrors)},
					{"Connect failed", fmt.Sprintf("%d", errStats.ConnectFailed)},
					{"Upstream timeout", fmt.Sprintf("%d", errStats.UpstreamTimed)},
				})
			}
			if cors, ok := report["correlations"].([]logs.UpstreamCorrelation); ok && len(cors) > 0 {
				fmt.Println()
				printGroup(st, "Correlations")
				rows := make([][]string, 0, len(cors))
				for _, c := range cors {
					rows = append(rows, []string{
						c.Upstream,
						fmt.Sprintf("%.1f%%", c.Access5xxPct),
						fmt.Sprintf("%d", c.Access502),
						fmt.Sprintf("%d", c.ErrorTotal),
					})
				}
				printTable(st, []int{20, 10, 8, 8}, []string{"UPSTREAM", "5XX %", "502", "ERRORS"}, rows)
			}
			if stats, ok := report["access"].(*logs.Stats); ok && len(stats.TopPaths) > 0 {
				fmt.Println()
				printGroup(st, fmt.Sprintf("Top paths (%d total)", stats.TotalRequests))
				rows := make([][]string, 0, len(stats.TopPaths))
				for _, e := range stats.TopPaths {
					rows = append(rows, []string{e.Key, fmt.Sprintf("%d", e.Count)})
				}
				printTable(st, []int{40, 10}, []string{"PATH", "COUNT"}, rows)
			}
			return nil
		},
	}
}
