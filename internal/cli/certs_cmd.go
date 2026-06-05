package cli

import (
	"fmt"
	"os"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/spf13/cobra"
)

func newCertsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "certs",
		Short: "Аудит SSL/TLS сертификатов (настройки: certs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}
			warnDays := cfg.Certs.WarnDays
			if warnDays == 0 {
				warnDays = 30
			}
			issues := analyzer.AuditCertificates(tree, warnDays, certVolumeMap(cfg))
			switch outputFormat(cfg) {
			case "json":
				return export.PrintJSON(issues)
			case "yaml":
				return export.PrintYAML(issues)
			}
			st := newStyler(cfg)
			if len(issues) == 0 {
				fmt.Println(st.ok("Проблем с сертификатами не найдено."))
				return nil
			}
			exitCode := 0
			for _, iss := range issues {
				fmt.Printf("[%s] %s: %s\n", st.severity(string(iss.Severity)), st.cyan(iss.Type), iss.Message)
				if iss.CertPath != "" {
					fmt.Printf("  cert: %s\n", st.gray(iss.CertPath))
				}
				if cfg.Certs.FailOnExpired && iss.Type == "cert_expired" {
					exitCode = 1
				}
			}
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}
}
