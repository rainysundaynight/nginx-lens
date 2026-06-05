package analyzer

import (
	"regexp"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Неиспользуемые переменные ----------
// set/map переменные без использования $var.

// UnusedVariable — неиспользуемая переменная.
type UnusedVariable struct {
	Name string `json:"name"`
}

var varRe = regexp.MustCompile(`\$[a-zA-Z0-9_]+`)

// FindUnusedVariables находит set/map переменные без использования.
func FindUnusedVariables(tree *parser.ConfigTree) []UnusedVariable {
	defined := make(map[string]struct{})
	used := make(map[string]struct{})

	for _, item := range Walk(tree) {
		if item.Node.Directive == "set" || item.Node.Directive == "map" {
			parts := strings.Fields(item.Node.Args)
			if len(parts) > 0 {
				defined[parts[0]] = struct{}{}
			}
		}
		for _, v := range varRe.FindAllString(item.Node.Args, -1) {
			used[v] = struct{}{}
		}
	}
	var unused []UnusedVariable
	for varName := range defined {
		if _, ok := used[varName]; !ok {
			unused = append(unused, UnusedVariable{Name: varName})
		}
	}
	return unused
}
