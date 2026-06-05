package cli

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Комплексная валидация для CI/CD (настройки: validate, defaults, output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireConfig()
			if err != nil {
				return err
			}
			st := newStyler(cfg)
			failOn := validateFailOn(cfg)
			results := map[string]interface{}{
				"syntax":   map[string]interface{}{"valid": false, "errors": []string{}},
				"analysis": map[string]interface{}{},
				"upstream": map[string]interface{}{"healthy": true},
				"dns":      map[string]interface{}{"resolved": true},
				"fail_on":  failOn,
			}
			exitCode := 0
			issueFails := func(sev analyzer.Severity) bool {
				return analyzer.IssueMeetsFailOn(sev, failOn)
			}

			if cfg.Policy.PolicyOnly {
				tree, err := parseConfigFromCfg(cfg)
				if err != nil {
					return err
				}
				printSection(st, "Policy-only режим")
				engine := policyEngineFromCfg(cfg)
				policyIssues := engine.Run(tree)
				results["policy"] = policyIssues
				for _, iss := range policyIssues {
					if issueFails(iss.Severity) {
						exitCode = 1
					}
				}
				fmt.Printf("Policy violations: %d (fail_on=%s)\n", len(policyIssues), failOn)
				switch outputFormat(cfg) {
				case "json":
					export.PrintJSON(results)
				case "yaml":
					export.PrintYAML(results)
				}
				if exitCode != 0 {
					os.Exit(exitCode)
				}
				return nil
			}

			if !cfg.Validate.SkipSyntax {
				printSection(st, "1. Проверка синтаксиса")
				valid, errors, synErr := checkNginxSyntax(cfg, cfg.Validate.SkipWarns)
				if synErr != nil {
					return synErr
				}
				results["syntax"] = map[string]interface{}{"valid": valid, "errors": errors}
				if valid {
					fmt.Println(st.ok("Синтаксис корректен"))
				} else {
					fmt.Println(st.fail("Ошибки синтаксиса"))
					for _, e := range errors {
						fmt.Println(" ", st.red(e))
					}
					exitCode = 1
				}
			}

			tree, err := parseConfigFromCfg(cfg)
			if err != nil {
				return err
			}

			if !cfg.Validate.SkipAnalysis {
				printSection(st, "2. Анализ проблем")
				result := analyzer.RunAnalysis(tree)
				exportData := export.FormatAnalyzeResults(result, analyzeFilter(cfg))
				if !cfg.Validate.SkipPolicy && len(cfg.Policy.Packs)+len(cfg.Policy.Rules) > 0 {
					engine := policyEngineFromCfg(cfg)
					for _, pi := range policyToAnalyzerIssues(engine.Run(tree)) {
						exportData.Issues = append(exportData.Issues, pi)
						exportData.Summary[string(pi.Severity)]++
					}
				}
				results["analysis"] = exportData
				for _, issue := range exportData.Issues {
					if issueFails(issue.Severity) {
						exitCode = 1
					}
				}
				fmt.Printf("Найдено issues: %d (high=%d medium=%d low=%d, fail_on=%s)\n",
					len(exportData.Issues),
					exportData.Summary["high"],
					exportData.Summary["medium"],
					exportData.Summary["low"],
					failOn)
			}

			if !cfg.Validate.SkipCerts && !cfg.Policy.PolicyOnly {
				printSection(st, "SSL/TLS сертификаты")
				warnDays := cfg.Certs.WarnDays
				if warnDays == 0 {
					warnDays = 30
				}
				certIssues := analyzer.AuditCertificates(tree, warnDays, certVolumeMap(cfg))
				results["certs"] = certIssues
				for _, c := range certIssues {
					if issueFails(c.Severity) {
						exitCode = 1
					}
				}
				fmt.Printf("Cert issues: %d\n", len(certIssues))
			}

			ups := tree.GetUpstreams()
			if !cfg.Validate.SkipUpstream {
				printSection(st, "3. Проверка upstream")
				health := upstream.CheckUpstreams(
					ups,
					cfg.Defaults.Timeout,
					cfg.Defaults.Retries,
					cfg.Defaults.Mode,
					cfg.Defaults.MaxWorkers,
				)
				results["upstream"] = map[string]interface{}{"servers": health}
				for _, servers := range health {
					for _, s := range servers {
						if !s.Healthy {
							exitCode = 1
						}
					}
				}
			}

			if !cfg.Validate.SkipDNS {
				printSection(st, "4. DNS resolve")
				setupDNSCache(cfg, false)
				resolved := upstream.ResolveUpstreams(ups, cfg.Defaults.MaxWorkers, true, cfg.Cache.TTL, "")
				results["dns"] = map[string]interface{}{"servers": resolved}
				for _, servers := range resolved {
					for _, s := range servers {
						if len(s.Resolved) == 0 {
							exitCode = 1
						}
					}
				}
			}

			switch outputFormat(cfg) {
			case "json":
				export.PrintJSON(results)
			case "yaml":
				export.PrintYAML(results)
			}

			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}
}

func checkSyntaxLocal(configPath, nginxPath string, skipWarns bool) (bool, []string) {
	cmd := exec.Command(nginxPath, "-t", "-c", configPath)
	out, err := cmd.CombinedOutput()
	combined := string(out)
	if err != nil {
		return false, []string{combined}
	}
	warnRe := regexp.MustCompile(`(?i)\[warn\]`)
	var warns []string
	for _, line := range strings.Split(combined, "\n") {
		if warnRe.MatchString(line) {
			warns = append(warns, strings.TrimSpace(line))
		}
	}
	if len(warns) > 0 && !skipWarns {
		return false, warns
	}
	return true, nil
}
