package analyzer

import (
	"regexp"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Проблемы rewrite ----------
// Циклы, конфликты и отсутствие флагов last/break.

// RewriteIssue — проблема с rewrite-директивой.
type RewriteIssue struct {
	Type    string       `json:"type"`
	Context *parser.Node `json:"context,omitempty"`
	Value   string       `json:"value"`
}

// FindRewriteIssues находит проблемы с rewrite-правилами.
func FindRewriteIssues(tree *parser.ConfigTree) []RewriteIssue {
	var issues []RewriteIssue
	type rewriteEntry struct {
		pattern string
		target  string
		context *parser.Node
		raw     string
	}
	var rewrites []rewriteEntry

	for _, item := range Walk(tree) {
		if item.Node.Directive != "rewrite" {
			continue
		}
		parts := strings.Fields(item.Node.Args)
		if len(parts) >= 2 {
			rewrites = append(rewrites, rewriteEntry{
				pattern: parts[0],
				target:  parts[1],
				context: item.Parent,
				raw:     item.Node.Args,
			})
		}
	}
	for _, r := range rewrites {
		if r.pattern == r.target {
			issues = append(issues, RewriteIssue{Type: "rewrite_cycle", Context: r.context, Value: r.raw})
		}
	}
	seen := make(map[string]string)
	for _, r := range rewrites {
		if prev, ok := seen[r.pattern]; ok && prev != r.target {
			issues = append(issues, RewriteIssue{
				Type:    "rewrite_conflict",
				Context: r.context,
				Value:   r.pattern + " -> " + prev + " и " + r.pattern + " -> " + r.target,
			})
		}
		seen[r.pattern] = r.target
	}
	flagRe := regexp.MustCompile(`\b(last|break|redirect|permanent)\b`)
	for _, item := range Walk(tree) {
		if item.Node.Directive == "rewrite" && !flagRe.MatchString(item.Node.Args) {
			issues = append(issues, RewriteIssue{Type: "rewrite_no_flag", Context: item.Parent, Value: item.Node.Args})
		}
	}
	return issues
}
