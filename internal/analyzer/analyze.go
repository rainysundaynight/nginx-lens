package analyzer

import "github.com/rainysundaynight/nginx-lens/internal/parser"

// ---------- Агрегация анализа ----------
// Сбор всех результатов статического анализа.

// AnalysisResult — полный результат статического анализа.
type AnalysisResult struct {
	LocationConflicts      []LocationConflict `json:"location_conflicts"`
	Duplicates             []Duplicate        `json:"duplicates"`
	EmptyBlocks            []EmptyBlock       `json:"empty_blocks"`
	Warnings               []Warning          `json:"warnings"`
	UnusedVariables        []UnusedVariable   `json:"unused_variables"`
	ListenConflicts        []ListenConflict   `json:"listen_conflicts"`
	RewriteIssues          []RewriteIssue     `json:"rewrite_issues"`
	DeadLocations          []DeadLocation     `json:"dead_locations"`
}

// RunAnalysis выполняет полный статический анализ конфигурации.
func RunAnalysis(tree *parser.ConfigTree) AnalysisResult {
	return AnalysisResult{
		LocationConflicts: FindLocationConflicts(tree),
		Duplicates:        FindDuplicateDirectives(tree),
		EmptyBlocks:       FindEmptyBlocks(tree),
		Warnings:          FindWarnings(tree),
		UnusedVariables:   FindUnusedVariables(tree),
		ListenConflicts:   FindListenServerNameConflicts(tree),
		RewriteIssues:     FindRewriteIssues(tree),
		DeadLocations:     FindDeadLocations(tree),
	}
}

// Issue — единая запись issue для экспорта.
type Issue struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Solution    string   `json:"solution"`
	Severity    Severity `json:"severity"`
	File        string   `json:"file,omitempty"`
	Line        int      `json:"line,omitempty"`
	FixHint     string   `json:"fix_hint,omitempty"`
}

// CollectIssues собирает все issues в единый список.
func CollectIssues(result AnalysisResult) []Issue {
	var issues []Issue
	add := func(issueType, desc, file string, line int) {
		meta, ok := DefaultIssueMeta[issueType]
		if !ok {
			meta = IssueMeta{"", SeverityLow, ""}
		}
		issues = append(issues, Issue{
			Type:        issueType,
			Description: desc,
			Solution:    meta.Solution,
			Severity:    meta.Severity,
			File:        file,
			Line:        line,
			FixHint:     meta.FixHint,
		})
	}
	for _, c := range result.LocationConflicts {
		add("location_conflict", "location: "+c.Location1+" ↔ "+c.Location2, c.Server.File, c.Server.Line)
	}
	for _, d := range result.Duplicates {
		add("duplicate_directive", d.Directive+" ("+d.Args+")", d.Block.File, d.Block.Line)
	}
	for _, e := range result.EmptyBlocks {
		add("empty_block", e.Block+" "+e.Arg, e.File, e.Line)
	}
	for _, w := range result.Warnings {
		file, line := "", 0
		if w.Context != nil {
			file, line = w.Context.File, w.Context.Line
		}
		add(w.Type, w.Value, file, line)
	}
	for _, v := range result.UnusedVariables {
		add("unused_variable", v.Name, "", 0)
	}
	for _, lc := range result.ListenConflicts {
		add("listen_servername_conflict", "listen/server_name conflict", lc.Server1.File, lc.Server1.Line)
	}
	for _, r := range result.RewriteIssues {
		file, line := "", 0
		if r.Context != nil {
			file, line = r.Context.File, r.Context.Line
		}
		add(r.Type, r.Value, file, line)
	}
	for _, d := range result.DeadLocations {
		add("dead_location", d.Location.Arg, d.Location.File, d.Location.Line)
	}
	return issues
}
