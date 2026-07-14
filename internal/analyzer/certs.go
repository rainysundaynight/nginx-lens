package analyzer

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- SSL/TLS аудит сертификатов ----------
// Пути — из ssl_certificate в дереве (nginx.conf + include/conf.d после parse / nginx -T).

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

// CertReadFile читает PEM по логическому пути из nginx.conf / conf.d.
type CertReadFile func(path string) ([]byte, error)

// AuditCertificates проверяет SSL-сертификаты из конфигурации.
func AuditCertificates(tree *parser.ConfigTree, warnDays int, volumeMap map[string]string) []CertIssue {
	return AuditCertificatesRead(tree, warnDays, defaultCertReader(volumeMap))
}

// AuditCertificatesRead — аудит с кастомным reader (docker exec / volume_map).
func AuditCertificatesRead(tree *parser.ConfigTree, warnDays int, readFile CertReadFile) []CertIssue {
	if readFile == nil {
		readFile = os.ReadFile
	}
	var issues []CertIssue
	seenCert := make(map[string]struct{})
	seenServerSSL := make(map[string]struct{})

	// Все ssl_certificate из основного конфига и include (в т.ч. conf.d/*.conf).
	for _, item := range Walk(tree) {
		if item.Node.Directive != "ssl_certificate" {
			continue
		}
		raw := strings.TrimSpace(strings.Split(item.Node.Args, " ")[0])
		if raw == "" || raw == "ssl_certificate" || strings.HasPrefix(raw, "$") {
			continue
		}
		sourceFile := item.Node.File
		if sourceFile == "" {
			sourceFile = certSourceFile(item)
		}
		certPath := resolveCertPath(raw, sourceFile)
		serverNames := serverNamesFromAncestors(item)
		key := certPath + "\x00" + serverNames
		if _, ok := seenCert[key]; ok {
			continue
		}
		seenCert[key] = struct{}{}
		issues = append(issues, checkCertFile(certPath, serverNames, sourceFile, warnDays, readFile)...)
	}

	for _, item := range Walk(tree) {
		if item.Node.Block != "server" {
			continue
		}
		hasCert := false
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			if sub.Node.Directive != "ssl_certificate" {
				continue
			}
			raw := strings.TrimSpace(strings.Split(sub.Node.Args, " ")[0])
			if raw != "" && raw != "ssl_certificate" && !strings.HasPrefix(raw, "$") {
				hasCert = true
				break
			}
		}
		sk := item.Node.File + "\x00" + fmt.Sprintf("%d", item.Node.Line)
		if _, ok := seenServerSSL[sk]; ok {
			continue
		}
		seenServerSSL[sk] = struct{}{}
		issues = append(issues, auditServerSSL(item, hasCert)...)
	}
	return issues
}

func defaultCertReader(volumeMap map[string]string) CertReadFile {
	return func(path string) ([]byte, error) {
		hostPath := path
		if mapped, ok := mapCertPath(volumeMap, path); ok {
			hostPath = mapped
		}
		return os.ReadFile(hostPath)
	}
}

// resolveCertPath приводит относительный путь ssl_certificate к абсолютному от файла конфига.
func resolveCertPath(certPath, sourceFile string) string {
	if certPath == "" {
		return certPath
	}
	if filepath.IsAbs(certPath) {
		return filepath.Clean(certPath)
	}
	if sourceFile == "" {
		return filepath.Clean(certPath)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), certPath))
}

func certSourceFile(item WalkItem) string {
	if item.Parent != nil && item.Parent.File != "" {
		return item.Parent.File
	}
	for i := len(item.Ancestors) - 1; i >= 0; i-- {
		if item.Ancestors[i] != nil && item.Ancestors[i].File != "" {
			return item.Ancestors[i].File
		}
	}
	return ""
}

func serverNamesFromAncestors(item WalkItem) string {
	server := nearestServer(item)
	if server == nil {
		return ""
	}
	var names []string
	for _, sub := range WalkNodes(server.Directives, server) {
		if sub.Node.Directive == "server_name" {
			names = append(names, strings.Fields(sub.Node.Args)...)
		}
	}
	return strings.Join(names, " ")
}

func nearestServer(item WalkItem) *parser.Node {
	if item.Parent != nil && item.Parent.Block == "server" {
		return item.Parent
	}
	for i := len(item.Ancestors) - 1; i >= 0; i-- {
		if item.Ancestors[i] != nil && item.Ancestors[i].Block == "server" {
			return item.Ancestors[i]
		}
	}
	return nil
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
	base := wildcard[2:]
	if host == base {
		return true
	}
	suffix := wildcard[1:]
	return strings.HasSuffix(host, suffix) && strings.Count(host, ".") >= strings.Count(suffix, ".")
}

func checkCertFile(path, serverNames, file string, warnDays int, readFile CertReadFile) []CertIssue {
	var issues []CertIssue
	if readFile == nil {
		readFile = os.ReadFile
	}
	data, err := readFile(path)
	if err != nil {
		issues = append(issues, CertIssue{
			Type: "cert_not_found", Severity: SeverityHigh,
			CertPath: path, Message: err.Error(), File: file,
			FixHint: "Проверьте путь ssl_certificate, volume_map или доступ docker exec",
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
