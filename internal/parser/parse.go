package parser

import (
	"fmt"
	"os/exec"
	"strings"
)

// ---------- Единая точка входа парсера ----------
// Режимы: expanded (nginx -T), regex, auto.

// ParseOptions — опции парсинга конфигурации.
type ParseOptions struct {
	Mode      string
	NginxPath string
}

// ParseConfig парсит nginx.conf согласно режиму.
func ParseConfig(path string, opts ParseOptions) (*ConfigTree, error) {
	mode := opts.Mode
	if mode == "" || mode == "auto" {
		if canRunExpanded(opts.NginxPath) {
			mode = "expanded"
		} else {
			mode = "regex"
		}
	}
	switch mode {
	case "expanded":
		tree, err := ParseExpanded(path, opts.NginxPath)
		if err != nil && opts.Mode == "auto" {
			return ParseNginxConfig(path)
		}
		return tree, err
	case "regex":
		return ParseNginxConfig(path)
	default:
		return nil, fmt.Errorf("неизвестный parser.mode: %s", mode)
	}
}

func canRunExpanded(nginxPath string) bool {
	if nginxPath == "" {
		nginxPath = "nginx"
	}
	_, err := exec.LookPath(nginxPath)
	return err == nil
}

// ParseLocationArg разбирает модификатор location.
func ParseLocationArg(raw string) (modifier, path string) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "="):
		return "=", strings.TrimSpace(strings.TrimPrefix(raw, "="))
	case strings.HasPrefix(raw, "^~"):
		return "^~", strings.TrimSpace(strings.TrimPrefix(raw, "^~"))
	case strings.HasPrefix(raw, "~*"):
		return "~*", strings.TrimSpace(strings.TrimPrefix(raw, "~*"))
	case strings.HasPrefix(raw, "~"):
		return "~", strings.TrimSpace(strings.TrimPrefix(raw, "~"))
	default:
		return "", raw
	}
}
