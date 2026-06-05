package config

import (
	"fmt"
	"strings"
)

// ---------- Валидация схемы config.yaml ----------
// Проверка обязательных полей и допустимых значений.

// ValidateSchema проверяет конфигурацию nginx-lens на корректность.
func ValidateSchema(cfg *Config) []string {
	var errs []string
	if cfg.Defaults.NginxConfigPath == "" {
		errs = append(errs, "defaults.nginx_config_path обязателен")
	}
	if cfg.Defaults.Timeout <= 0 {
		errs = append(errs, "defaults.timeout должен быть > 0")
	}
	mode := cfg.Parser.Mode
	if mode != "" && mode != "auto" && mode != "expanded" && mode != "regex" {
		errs = append(errs, fmt.Sprintf("parser.mode недопустим: %s (auto|expanded|regex)", mode))
	}
	out := cfg.Output.Format
	if out != "" {
		allowed := map[string]struct{}{"table": {}, "json": {}, "yaml": {}, "csv": {}, "prometheus": {}}
		if _, ok := allowed[out]; !ok {
			errs = append(errs, fmt.Sprintf("output.format недопустим: %s", out))
		}
	}
	sev := cfg.Analyze.MinSeverity
	if sev != "" && sev != "low" && sev != "medium" && sev != "high" {
		errs = append(errs, fmt.Sprintf("analyze.min_severity недопустим: %s", sev))
	}
	for _, p := range cfg.Policy.Packs {
		if p != "" && !isKnownPack(p) {
			errs = append(errs, fmt.Sprintf("policy.packs: неизвестный pack %q", p))
		}
	}
	dm := strings.ToLower(cfg.Docker.Enabled)
	if dm != "" && dm != "auto" && dm != "true" && dm != "false" {
		errs = append(errs, fmt.Sprintf("docker.enabled недопустим: %s (auto|true|false)", cfg.Docker.Enabled))
	}
	for i, r := range cfg.Policy.Rules {
		if r.ID == "" {
			errs = append(errs, fmt.Sprintf("policy.rules[%d]: id обязателен", i))
		}
		if r.Match == "" {
			errs = append(errs, fmt.Sprintf("policy.rules[%d]: match обязателен", i))
		}
	}
	return errs
}

func isKnownPack(name string) bool {
	switch name {
	case "security-baseline", "mozilla-ssl",
		"performance", "performance-baseline",
		"caching", "rate-limit":
		return true
	default:
		return strings.HasPrefix(name, "custom-")
	}
}
