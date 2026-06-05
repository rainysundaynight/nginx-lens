package nginxinfo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ---------- Парсинг nginx -V ----------
// Версия и список модулей сборки для snapshot и Hub UI.

// BuildInfo — сведения о сборке nginx.
type BuildInfo struct {
	Version        string   `json:"version"`
	BuiltBy        string   `json:"built_by,omitempty"`
	OpenSSL        string   `json:"openssl,omitempty"`
	TLS            string   `json:"tls,omitempty"`
	ConfigureArgs  string   `json:"configure_args,omitempty"`
	StaticModules  []string `json:"static_modules"`
	DynamicModules []string `json:"dynamic_modules"`
}

var (
	reNginxVersion = regexp.MustCompile(`nginx version:\s*nginx/([^\s]+)`)
	reBuiltBy      = regexp.MustCompile(`(?m)^built by (.+)$`)
	reOpenSSL      = regexp.MustCompile(`(?m)^built with (.+OpenSSL[^\n]*)`)
	reTLS          = regexp.MustCompile(`(?m)^(TLS .+)$`)
	reConfigure    = regexp.MustCompile(`(?m)^configure arguments:\s*(.+)$`)
	reWithFlag     = regexp.MustCompile(`--with-([\w-]+)`)
	reAddDynamic   = regexp.MustCompile(`--add-dynamic-module=([^\s'"]+)`)
	reAddModule    = regexp.MustCompile(`--add-module=([^\s'"]+)`)
)

// RunVersion выполняет nginx -V и парсит вывод.
func RunVersion(nginxPath string) (*BuildInfo, error) {
	if nginxPath == "" {
		nginxPath = "nginx"
	}
	cmd := exec.Command(nginxPath, "-V")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return nil, fmt.Errorf("nginx -V: %w", err)
		}
	}
	return ParseVersionOutput(string(out))
}

// ParseVersionOutput разбирает combined stdout/stderr nginx -V.
func ParseVersionOutput(raw string) (*BuildInfo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("пустой вывод nginx -V")
	}
	info := &BuildInfo{}
	if m := reNginxVersion.FindStringSubmatch(raw); len(m) > 1 {
		info.Version = m[1]
	}
	if m := reBuiltBy.FindStringSubmatch(raw); len(m) > 1 {
		info.BuiltBy = strings.TrimSpace(m[1])
	}
	if m := reOpenSSL.FindStringSubmatch(raw); len(m) > 1 {
		info.OpenSSL = strings.TrimSpace(m[1])
	}
	if m := reTLS.FindStringSubmatch(raw); len(m) > 1 {
		info.TLS = strings.TrimSpace(m[1])
	}
	if m := reConfigure.FindStringSubmatch(raw); len(m) > 1 {
		info.ConfigureArgs = strings.TrimSpace(m[1])
		info.StaticModules, info.DynamicModules = extractModules(info.ConfigureArgs)
	}
	if info.Version == "" && info.ConfigureArgs == "" {
		return nil, fmt.Errorf("не удалось разобрать nginx -V")
	}
	return info, nil
}

func extractModules(args string) (static []string, dynamic []string) {
	seen := make(map[string]struct{})
	for _, m := range reWithFlag.FindAllStringSubmatch(args, -1) {
		name := normalizeModuleName(m[1])
		if name == "" || isBuildFlag(name) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		static = append(static, name)
	}
	dynSeen := make(map[string]struct{})
	for _, re := range []*regexp.Regexp{reAddDynamic, reAddModule} {
		for _, m := range re.FindAllStringSubmatch(args, -1) {
			name := normalizeModulePath(m[1])
			if name == "" {
				continue
			}
			if _, ok := dynSeen[name]; ok {
				continue
			}
			dynSeen[name] = struct{}{}
			dynamic = append(dynamic, name)
		}
	}
	return static, dynamic
}

func normalizeModuleName(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), "_module")
}

func isBuildFlag(name string) bool {
	switch name {
	case "cc-opt", "ld-opt", "debug", "pcre", "pcre-jit", "pcre2", "pcre2-jit", "ipv6", "file-aio", "threads", "compat":
		return true
	}
	return strings.HasPrefix(name, "cc-") || strings.HasPrefix(name, "ld-")
}

func normalizeModulePath(p string) string {
	p = strings.Trim(p, `"'`)
	p = strings.TrimSuffix(p, "/")
	return filepath.Base(p)
}
