package analyzer

import "strings"

// ---------- Фильтрация issues ----------
// Severity filter для analyze и validate exit codes.

// Severity — уровень серьёзности проблемы.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// FilterOptions — опции фильтрации issues.
type FilterOptions struct {
	SkipLow     bool
	SkipMedium  bool
	SkipTypes   map[string]struct{}
	MinSeverity Severity
}

// ShouldIncludeIssue проверяет, нужно ли включать issue в вывод.
func ShouldIncludeIssue(issueType string, severity Severity, opts FilterOptions) bool {
	if opts.SkipTypes != nil {
		if _, skip := opts.SkipTypes[issueType]; skip {
			return false
		}
	}
	if opts.SkipLow && severity == SeverityLow {
		return false
	}
	if opts.SkipMedium && severity == SeverityMedium {
		return false
	}
	return severityMeetsMin(severity, opts.MinSeverity)
}

// IssueAffectsExit определяет, влияет ли issue на exit code validate.
func IssueAffectsExit(severity Severity, failOnLow bool) bool {
	return IssueMeetsFailOn(severity, failOnThreshold(failOnLow))
}

// IssueMeetsFailOn сравнивает severity с порогом fail_on (high|medium|low).
func IssueMeetsFailOn(severity Severity, failOn string) bool {
	failOn = strings.ToLower(strings.TrimSpace(failOn))
	if failOn == "" || failOn == "high" {
		return severity == SeverityHigh
	}
	switch failOn {
	case "medium":
		return severity == SeverityHigh || severity == SeverityMedium
	case "low":
		return true
	default:
		return severity == SeverityHigh
	}
}

func failOnThreshold(failOnLow bool) string {
	if failOnLow {
		return "low"
	}
	return "high"
}

func severityMeetsMin(sev, min Severity) bool {
	order := map[Severity]int{SeverityLow: 0, SeverityMedium: 1, SeverityHigh: 2}
	return order[sev] >= order[min]
}

// IssueMeta — описание, severity и fix hint для типа issue.
type IssueMeta struct {
	Solution string
	Severity Severity
	FixHint  string
}

// DefaultIssueMeta — метаданные для всех типов issues.
var DefaultIssueMeta = map[string]IssueMeta{
	"location_conflict":          {"Возможное пересечение location. Проверьте порядок и типы location.", SeverityMedium, "# Используйте = / ^~ / ~ для явного приоритета"},
	"duplicate_directive":          {"Оставьте одну директиву в блоке.", SeverityMedium, ""},
	"empty_block":                  {"Удалите или заполните пустой блок.", SeverityLow, ""},
	"proxy_pass_no_scheme":         {"Добавьте http:// или https:// в proxy_pass.", SeverityMedium, "proxy_pass http://backend;"},
	"autoindex_on":                 {"Отключите autoindex.", SeverityMedium, "autoindex off;"},
	"if_block":                     {"Избегайте if внутри location.", SeverityMedium, ""},
	"server_tokens_on":             {"Отключите server_tokens.", SeverityHigh, "server_tokens off;"},
	"server_tokens_off_missing":    {"server_tokens off не задан (по умолчанию nginx раскрывает версию).", SeverityHigh, "server_tokens off;"},
	"ssl_missing":                  {"Укажите SSL-сертификат/ключ.", SeverityHigh, "ssl_certificate /path/cert.pem;\nssl_certificate_key /path/key.pem;"},
	"ssl_protocols_weak":           {"Отключите устаревшие TLS.", SeverityHigh, "ssl_protocols TLSv1.2 TLSv1.3;"},
	"ssl_ciphers_weak":             {"Используйте современные шифры.", SeverityHigh, "ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;"},
	"listen_443_no_ssl":            {"Добавьте ssl к listen 443.", SeverityHigh, "listen 443 ssl;"},
	"listen_443_no_http2":          {"Добавьте http2 к listen 443.", SeverityLow, "listen 443 ssl http2;"},
	"no_limit_req_conn":            {"Добавьте limit_req/limit_conn.", SeverityMedium, "limit_req zone=one burst=20 nodelay;"},
	"missing_security_header":      {"Добавьте security-заголовок.", SeverityMedium, `add_header X-Frame-Options "SAMEORIGIN" always;`},
	"deprecated":                   {"Замените устаревшую директиву.", SeverityMedium, ""},
	"limit_too_small":              {"Увеличьте лимит.", SeverityMedium, ""},
	"limit_too_large":              {"Уменьшите лимит.", SeverityMedium, ""},
	"unused_variable":              {"Удалите переменную.", SeverityLow, ""},
	"listen_servername_conflict":   {"Исправьте listen/server_name.", SeverityHigh, ""},
	"rewrite_cycle":                {"Проверьте rewrite на циклы.", SeverityHigh, ""},
	"rewrite_conflict":             {"Проверьте порядок rewrite.", SeverityMedium, ""},
	"rewrite_no_flag":              {"Добавьте last/break к rewrite.", SeverityLow, "rewrite ^/old(/.*)$ /new$1 last;"},
	"dead_location":                {"Удалите или используйте location.", SeverityLow, ""},
	"missing_nginx_module":         {"Пересоберите nginx с нужным модулем.", SeverityHigh, ""},
}
