package agent

import (
	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/export"
	"github.com/rainysundaynight/nginx-lens/internal/logs"
	"github.com/rainysundaynight/nginx-lens/internal/nginxinfo"
	"github.com/rainysundaynight/nginx-lens/internal/nginxload"
	"github.com/rainysundaynight/nginx-lens/internal/policy"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

// ---------- Сбор snapshot ----------
// Агрегация analyze + upstream health + blast-radius для agent API.

// Snapshot — полный снимок состояния nginx-хоста.
type Snapshot struct {
	ConfigPath        string                                 `json:"config_path"`
	Cache             map[string]interface{}                 `json:"cache"`
	DynamicUpstream   config.DynamicUpstreamConfig           `json:"dynamic_upstream"`
	Analyze           export.AnalyzeExport                   `json:"analyze"`
	Upstreams         map[string][]string                    `json:"upstreams"`
	UpstreamsHealth   map[string][]upstream.ServerHealth     `json:"upstreams_health"`
	DependencyGraph   map[string][]analyzer.BlastRadiusEntry   `json:"dependency_graph"`
	StreamGraph       upstream.StreamGraph                   `json:"stream_graph,omitempty"`
	Score             analyzer.ScoreReport                   `json:"score"`
	PolicyIssues      []policy.Issue                         `json:"policy_issues"`
	CertIssues        []analyzer.CertIssue                   `json:"cert_issues"`
	CertTimeline      []analyzer.CertTimelineEntry           `json:"cert_timeline,omitempty"`
	ErrorStats        *logs.ErrorStats                       `json:"error_stats,omitempty"`
	AccessStats       *logs.AccessSnapshot                   `json:"access_stats,omitempty"`
	LogCorrelations   []logs.UpstreamCorrelation             `json:"log_correlations,omitempty"`
	NginxBuild        *nginxinfo.BuildInfo                   `json:"nginx_build,omitempty"`
	ModuleGroups      nginxinfo.ModuleGroups                 `json:"module_groups,omitempty"`
	ModuleIssues      []nginxinfo.ModuleIssue                `json:"module_issues,omitempty"`
}

// CollectSnapshot собирает JSON-сериализуемый snapshot конфигурации.
func CollectSnapshot() (*Snapshot, error) {
	loader := config.Get()
	if err := config.RequireConfigFile(loader); err != nil {
		return nil, err
	}
	cfg := loader.Config
	tree, configPath, err := nginxload.BuildTree(cfg)
	if err != nil {
		return nil, err
	}

	result := analyzer.RunAnalysis(tree)
	filter := analyzer.FilterOptions{MinSeverity: analyzer.SeverityLow}
	analyzeExport := export.FormatAnalyzeResults(result, filter)

	engine := policy.NewEngine(cfg.Policy.Packs, policyRulesFromCfg(cfg))
	policyIssues := engine.Run(tree)
	for _, pi := range policyIssues {
		export.AppendIssue(&analyzeExport, issueFromPolicy(pi), filter)
	}

	upstreams := tree.GetUpstreams()
	defaults := cfg.Defaults
	streamGraph := upstream.BuildStreamGraph(tree)
	var health map[string][]upstream.ServerHealth
	if !cfg.Upstreams.SkipHealth {
		streamTCP := upstream.StreamReferencedNames(tree)
		health = upstream.CheckUpstreamsMixed(
			upstreams,
			streamTCP,
			defaults.Timeout,
			defaults.Retries,
			defaults.Mode,
			defaults.MaxWorkers,
		)
	}

	warnDays := cfg.Certs.WarnDays
	if warnDays == 0 {
		warnDays = 30
	}
	certIssues := analyzer.AuditCertificates(tree, warnDays, cfg.Docker.VolumeMap)
	for _, c := range certIssues {
		export.AppendIssue(&analyzeExport, issueFromCert(c), filter)
	}

	depGraph := analyzer.BuildDependencyGraph(tree)

	tailLines := cfg.Logs.TailLines
	if tailLines <= 0 {
		tailLines = 50000
	}

	var errorStats *logs.ErrorStats
	if cfg.Logs.ErrorPath != "" {
		if raw, err := nginxload.ReadLogTail(cfg, cfg.Logs.ErrorPath, tailLines); err == nil && raw != "" {
			errorStats, _ = logs.ParseErrorLogContent(raw)
		}
	}

	var accessStats *logs.AccessSnapshot
	if cfg.Logs.Path != "" {
		if raw, err := nginxload.ReadLogTail(cfg, cfg.Logs.Path, tailLines); err == nil && raw != "" {
			accessStats, _ = logs.ParseAccessSnapshotContent(raw, cfg.Logs.Since, cfg.Logs.Until, cfg.Logs.Status, cfg.Logs.FormatRegex)
		}
	}

	nginxBuild := nginxinfo.CollectBuildInfo(cfg)
	moduleIssues := nginxinfo.CheckDirectiveModules(tree, nginxBuild)
	for _, mi := range moduleIssues {
		if mi.Module == "nginx_build" {
			continue
		}
		export.AppendIssue(&analyzeExport, analyzer.Issue{
			Type: "missing_nginx_module", Description: mi.Message, Severity: analyzer.SeverityHigh,
			Solution: "Пересоберите nginx с нужным модулем или удалите директиву.",
			FixHint:  "Пересоберите nginx с модулем " + mi.Module + " или удалите директиву " + mi.Directive,
		}, filter)
	}

	score := analyzer.ComputeScoreFromIssues(analyzeExport.Issues, 0)

	correlations := logs.BuildCorrelations(accessStats, errorStats, depGraph, upstreams, streamGraph)

	return &Snapshot{
		ConfigPath: configPath,
		Cache: map[string]interface{}{
			"enabled": cfg.Cache.Enabled,
			"ttl":     cfg.Cache.TTL,
		},
		DynamicUpstream: cfg.DynamicUpstream,
		Analyze:         analyzeExport,
		Upstreams:       upstreams,
		UpstreamsHealth: health,
		DependencyGraph: depGraph,
		StreamGraph:     streamGraph,
		Score:           score,
		PolicyIssues:    policyIssues,
		CertIssues:      certIssues,
		CertTimeline:    analyzer.BuildCertTimeline(certIssues),
		ErrorStats:      errorStats,
		AccessStats:     accessStats,
		LogCorrelations: correlations,
		NginxBuild:      nginxBuild,
		ModuleGroups:    nginxinfo.GroupModules(nginxBuild),
		ModuleIssues:    moduleIssues,
	}, nil
}

func policyRulesFromCfg(cfg config.Config) []policy.CustomRule {
	var rules []policy.CustomRule
	for _, r := range cfg.Policy.Rules {
		rules = append(rules, policy.CustomRule{
			ID: r.ID, Match: r.Match, Severity: r.Severity,
			Message: r.Message, FixHint: r.FixHint,
		})
	}
	return rules
}

func issueFromPolicy(pi policy.Issue) analyzer.Issue {
	iss := analyzer.Issue{
		Type: pi.Type, Description: pi.Message, Severity: pi.Severity,
		File: pi.File, Line: pi.Line, FixHint: pi.FixHint,
	}
	if meta, ok := analyzer.DefaultIssueMeta[pi.Type]; ok {
		iss.Solution = meta.Solution
	} else {
		iss.Solution = pi.Message
	}
	return iss
}

func issueFromCert(c analyzer.CertIssue) analyzer.Issue {
	return analyzer.Issue{
		Type: c.Type, Description: c.Message, Severity: c.Severity,
		File: c.File, FixHint: c.FixHint, Solution: c.Message,
	}
}
