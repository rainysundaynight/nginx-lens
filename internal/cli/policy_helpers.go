package cli

import (
	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/policy"
)

func policyEngineFromCfg(cfg config.Config) *policy.Engine {
	var rules []policy.CustomRule
	for _, r := range cfg.Policy.Rules {
		rules = append(rules, policy.CustomRule{
			ID: r.ID, Match: r.Match, Severity: r.Severity,
			Message: r.Message, FixHint: r.FixHint,
		})
	}
	return policy.NewEngine(cfg.Policy.Packs, rules)
}

func policyToAnalyzerIssues(pi []policy.Issue) []analyzer.Issue {
	var issues []analyzer.Issue
	for _, p := range pi {
		issues = append(issues, analyzer.Issue{
			Type: p.Type, Description: p.Message, Solution: p.FixHint,
			Severity: p.Severity, File: p.File, Line: p.Line, FixHint: p.FixHint,
		})
	}
	return issues
}
