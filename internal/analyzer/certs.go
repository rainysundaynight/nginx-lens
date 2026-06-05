package analyzer

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- SSL/TLS аудит сертификатов ----------
// Проверка expiry, self-signed, hostname mismatch.

// CertIssue — проблема с сертификатом.
type CertIssue struct {
	Type        string    `json:"type"`
	Severity    Severity  `json:"severity"`
	CertPath    string    `json:"cert_path"`
	ServerName  string    `json:"server_name,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	DaysLeft    int       `json:"days_left,omitempty"`
	Message     string    `json:"message"`
	FixHint     string    `json:"fix_hint,omitempty"`
	File        string    `json:"file,omitempty"`
}

// AuditCertificates проверяет SSL-сертификаты из конфигурации.
func AuditCertificates(tree *parser.ConfigTree, warnDays int, volumeMap map[string]string) []CertIssue {
	var issues []CertIssue
	seen := make(map[string]struct{})

	for _, item := range Walk(tree) {
		if item.Node.Block != "server" {
			continue
		}
		var certPath, serverNames string
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			if sub.Node.Directive == "ssl_certificate" {
				certPath = strings.TrimSpace(strings.Split(sub.Node.Args, " ")[0])
			}
			if sub.Node.Directive == "server_name" {
				serverNames = sub.Node.Args
			}
		}
		if certPath == "" || certPath == "ssl_certificate" {
			continue
		}
		key := certPath + "\x00" + serverNames
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		hostPath := certPath
		if mapped, ok := mapCertPath(volumeMap, certPath); ok {
			hostPath = mapped
		}
		issues = append(issues, checkCertFile(hostPath, serverNames, item.Node.File, warnDays)...)
		issues = append(issues, auditServerSSL(item, certPath != "")...)
	}
	return issues
}

// CertTimelineEntry — точка таймлайна истечения сертификата.
type CertTimelineEntry struct {
	CertPath   string    `json:"cert_path"`
	ServerName string    `json:"server_name"`
	ExpiresAt  time.Time `json:"expires_at"`
	DaysLeft   int       `json:"days_left"`
	Severity   Severity  `json:"severity"`
}

// BuildCertTimeline строит таймлайн из cert issues.
func BuildCertTimeline(issues []CertIssue) []CertTimelineEntry {
	var timeline []CertTimelineEntry
	seen := make(map[string]struct{})
	for _, c := range issues {
		if c.ExpiresAt.IsZero() {
			continue
		}
		key := c.CertPath + "\x00" + c.ServerName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		timeline = append(timeline, CertTimelineEntry{
			CertPath: c.CertPath, ServerName: c.ServerName,
			ExpiresAt: c.ExpiresAt, DaysLeft: c.DaysLeft, Severity: c.Severity,
		})
	}
	return timeline
}

func auditServerSSL(item WalkItem, hasCert bool) []CertIssue {
	var issues []CertIssue
	hasSSLListen, hasHSTS, hasStapling := false, false, false
	var serverNames string
	for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
		switch sub.Node.Directive {
		case "listen":
			if strings.Contains(sub.Node.Args, "ssl") {
				hasSSLListen = true
			}
		case "server_name":
			serverNames = sub.Node.Args
		case "ssl_stapling":
			if strings.TrimSpace(sub.Node.Args) == "on" {
				hasStapling = true
			}
		case "add_header":
			if strings.Contains(sub.Node.Args, "Strict-Transport-Security") {
				hasHSTS = true
			}
		}
	}
	if hasCert && hasSSLListen && !hasStapling {
		issues = append(issues, CertIssue{
			Type: "ocsp_stapling_off", Severity: SeverityLow,
			File: item.Node.File, ServerName: serverNames,
			Message: "ssl_stapling не включён на HTTPS server",
			FixHint: "ssl_stapling on;\nssl_trusted_certificate /path/chain.pem;",
		})
	}
	if hasCert && hasSSLListen && !hasHSTS {
		issues = append(issues, CertIssue{
			Type: "hsts_missing", Severity: SeverityMedium,
			File: item.Node.File, ServerName: serverNames,
			Message: "отсутствует HSTS (Strict-Transport-Security)",
			FixHint: `add_header Strict-Transport-Security "max-age=31536000" always;`,
		})
	}
	for _, sn := range strings.Fields(serverNames) {
		if strings.HasPrefix(sn, "*.") {
			continue
		}
		for _, other := range strings.Fields(serverNames) {
			if strings.HasPrefix(other, "*.") && !hostMatchesWildcard(sn, other) {
				issues = append(issues, CertIssue{
					Type: "wildcard_server_name_hint", Severity: SeverityLow,
					File: item.Node.File, ServerName: sn,
					Message: fmt.Sprintf("server_name %s не покрывается wildcard %s", sn, other),
					FixHint: "Добавьте явный server_name или используйте сертификат с SAN",
				})
			}
		}
	}
	return issues
}

func hostMatchesWildcard(host, wildcard string) bool {
	if !strings.HasPrefix(wildcard, "*.") {
		return host == wildcard
	}
	suffix := wildcard[1:]
	return strings.HasSuffix(host, suffix) && strings.Count(host, ".") >= strings.Count(suffix, ".")
}

func checkCertFile(path, serverNames, file string, warnDays int) []CertIssue {
	var issues []CertIssue
	data, err := os.ReadFile(path)
	if err != nil {
		issues = append(issues, CertIssue{
			Type: "cert_not_found", Severity: SeverityHigh,
			CertPath: path, Message: err.Error(), File: file,
			FixHint: "Проверьте путь ssl_certificate и права доступа",
		})
		return issues
	}
	certs := parsePEMCerts(data)
	if len(certs) == 0 {
		issues = append(issues, CertIssue{
			Type: "cert_invalid_pem", Severity: SeverityHigh,
			CertPath: path, Message: "невалидный PEM", File: file,
		})
		return issues
	}
	cert := certs[0]
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	if now.After(cert.NotAfter) {
		issues = append(issues, CertIssue{
			Type: "cert_expired", Severity: SeverityHigh,
			CertPath: path, ServerName: serverNames, ExpiresAt: cert.NotAfter, DaysLeft: daysLeft,
			Message: fmt.Sprintf("сертификат истёк %s", cert.NotAfter.Format("2006-01-02")),
			FixHint: "Обновите сертификат и выполните nginx -s reload", File: file,
		})
	} else if daysLeft <= warnDays {
		issues = append(issues, CertIssue{
			Type: "cert_expiring", Severity: SeverityMedium,
			CertPath: path, ServerName: serverNames, ExpiresAt: cert.NotAfter, DaysLeft: daysLeft,
			Message: fmt.Sprintf("истекает через %d дней", daysLeft),
			FixHint: "Запланируйте обновление сертификата", File: file,
		})
	}
	if cert.Issuer.String() == cert.Subject.String() {
		issues = append(issues, CertIssue{
			Type: "cert_self_signed", Severity: SeverityLow,
			CertPath: path, Message: "self-signed сертификат", File: file,
		})
	} else if len(certs) == 1 {
		roots, err := x509.SystemCertPool()
		if err == nil && roots != nil {
			if _, err := cert.Verify(x509.VerifyOptions{Roots: roots}); err != nil {
				issues = append(issues, CertIssue{
					Type: "cert_chain_incomplete", Severity: SeverityHigh,
					CertPath: path, Message: "неполная цепочка: добавьте intermediate в ssl_certificate",
					FixHint: "Объедините leaf + intermediate в один PEM-файл ssl_certificate", File: file,
				})
			}
		}
	}
	for _, sn := range strings.Fields(serverNames) {
		if sn == "_" || sn == "" {
			continue
		}
		if err := cert.VerifyHostname(sn); err != nil {
			if strings.HasPrefix(sn, "*.") {
				continue
			}
			hasWildcardSAN := false
			for _, n := range cert.DNSNames {
				if strings.HasPrefix(n, "*.") && hostMatchesWildcard(sn, n) {
					hasWildcardSAN = true
					break
				}
			}
			if !hasWildcardSAN {
				issues = append(issues, CertIssue{
					Type: "cert_hostname_mismatch", Severity: SeverityMedium,
					CertPath: path, ServerName: sn,
					Message: fmt.Sprintf("CN/SAN не покрывает %s", sn), File: file,
				})
			}
		}
	}
	if len(cert.DNSNames) > 0 {
		for _, san := range cert.DNSNames {
			if !strings.HasPrefix(san, "*.") {
				continue
			}
			matched := false
			for _, sn := range strings.Fields(serverNames) {
				if hostMatchesWildcard(sn, san) || sn == san {
					matched = true
					break
				}
			}
			if !matched && serverNames != "" {
				issues = append(issues, CertIssue{
					Type: "wildcard_cert_unused", Severity: SeverityLow,
					CertPath: path, ServerName: serverNames,
					Message: fmt.Sprintf("сертификат wildcard %s, но server_name не использует этот паттерн", san),
					FixHint: "Добавьте поддомены в server_name или используйте точное имя в сертификате",
					File: file,
				})
			}
		}
	}
	return issues
}

func mapCertPath(volumeMap map[string]string, path string) (string, bool) {
	if len(volumeMap) == 0 {
		return "", false
	}
	longest := ""
	host := ""
	for prefix, h := range volumeMap {
		if strings.HasPrefix(path, prefix) && len(prefix) > len(longest) {
			longest = prefix
			host = h
		}
	}
	if longest == "" {
		return "", false
	}
	suffix := strings.TrimPrefix(path, longest)
	return strings.TrimRight(host, "/") + suffix, true
}

func parsePEMCerts(data []byte) []*x509.Certificate {
	var certs []*x509.Certificate
	rest := data
	for len(rest) > 0 {
		block, remain := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remain
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err == nil {
			certs = append(certs, c)
		}
	}
	return certs
}
