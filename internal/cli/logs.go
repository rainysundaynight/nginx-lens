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

			if snap, ok := report["access_snapshot"].(*logs.AccessSnapshot); ok {
				fmt.Printf("Access: %d req, RPS %.2f, 404=%d, 5xx=%d, p95=%.1fms\n",
					snap.TotalRequests, snap.RPS, snap.Status404, snap.Status5xx, snap.P95Latency)
			}
			if errStats, ok := report["error"].(*logs.ErrorStats); ok {
				fmt.Printf("Error: total=%d connect=%d timeout=%d\n",
					errStats.TotalErrors, errStats.ConnectFailed, errStats.UpstreamTimed)
			}
			if cors, ok := report["correlations"].([]logs.UpstreamCorrelation); ok && len(cors) > 0 {
				fmt.Println("\nCorrelations (upstream):")
				for _, c := range cors {
					fmt.Printf("  %s: access 5xx %.1f%% (%d), errors %d\n",
						c.Upstream, c.Access5xxPct, c.Access502, c.ErrorTotal)
				}
			}
			if stats, ok := report["access"].(*logs.Stats); ok {
				fmt.Printf("\nTop paths (%d total):\n", stats.TotalRequests)
				for _, e := range stats.TopPaths {
					fmt.Printf("  %s: %d\n", e.Key, e.Count)
				}
			}
			return nil
		},
	}
}
