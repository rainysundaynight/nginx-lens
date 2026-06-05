package cli

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/k8s"
	"github.com/spf13/cobra"
)

func newIngressAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ingress-audit",
		Short: "Аудит K8s Ingress vs nginx server_name (настройки: k8s.manifests_path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			if err := requireNonEmpty("k8s.manifests_path", cfg.K8s.ManifestsPath); err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			var serverNames []string
			for _, item := range analyzer.Walk(tree) {
				if item.Node.Directive == "server_name" {
					serverNames = append(serverNames, strings.Fields(item.Node.Args)...)
				}
			}
			issues, rules := k8s.AuditIngressManifests(cfg.K8s.ManifestsPath, k8s.CollectServerNames(serverNames))
			out := map[string]interface{}{"issues": issues, "rules": rules}
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(out)
			case "yaml":
				return export.PrintYAML(out)
			}
			st := newStyler(cfg)
			printSection(st, "Ingress audit")
			printSummary(st, "Rules: %d, issues: %d", len(rules), len(issues))
			if len(issues) == 0 {
				printEmptyOK(st, "Расхождений Ingress и nginx не найдено.")
				return nil
			}
			rows := make([][]string, 0, len(issues))
			for _, iss := range issues {
				rows = append(rows, []string{iss.Type, iss.Host, iss.Path, iss.Message})
			}
			printTable(st, []int{12, 24, 16, 0}, []string{"TYPE", "HOST", "PATH", "MESSAGE"}, rows)
			return nil
		},
	}
}
