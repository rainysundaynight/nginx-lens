package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/config"
)

// ---------- Цветной вывод в терминал ----------
// Управляется output.colors и NO_COLOR.

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
	ansiGray   = "\033[90m"
)

type styler struct {
	enabled bool
}

func newStyler(cfg config.Config) styler {
	on := cfg.Output.Colors
	if os.Getenv("NO_COLOR") != "" {
		on = false
	}
	return styler{enabled: on}
}

func (s styler) wrap(code, text string) string {
	if !s.enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}

func (s styler) bold(t string) string   { return s.wrap(ansiBold, t) }
func (s styler) red(t string) string   { return s.wrap(ansiRed, t) }
func (s styler) green(t string) string { return s.wrap(ansiGreen, t) }
func (s styler) yellow(t string) string { return s.wrap(ansiYellow, t) }
func (s styler) cyan(t string) string  { return s.wrap(ansiCyan, t) }
func (s styler) blue(t string) string  { return s.wrap(ansiBlue, t) }
func (s styler) gray(t string) string  { return s.wrap(ansiGray, t) }

func (s styler) ok(t string) string    { return s.green("✓ " + t) }
func (s styler) fail(t string) string  { return s.red("✗ " + t) }
func (s styler) header(t string) string { return s.bold(s.cyan(t)) }

func (s styler) severity(sev string) string {
	switch strings.ToLower(sev) {
	case "high":
		return s.red(sev)
	case "medium":
		return s.yellow(sev)
	case "low":
		return s.blue(sev)
	default:
		return sev
	}
}

func (s styler) diffType(t string) string {
	switch t {
	case "added":
		return s.green("+" + t)
	case "removed":
		return s.red("-" + t)
	case "changed":
		return s.yellow("~" + t)
	default:
		return t
	}
}

func (s styler) pathParts(parts []string) string {
	return strings.Join(parts, " → ")
}

func (s styler) println(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}
