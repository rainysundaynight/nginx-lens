package analyzer

// ---------- Config Score ----------
// Рейтинг готовности конфигурации 0-100.

// ScoreCategory — оценка по категории.
type ScoreCategory struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Weight  float64 `json:"weight"`
	Issues  int     `json:"issues"`
}

// ScoreReport — итоговый рейтинг.
type ScoreReport struct {
	Total       float64         `json:"total"`
	Categories  []ScoreCategory `json:"categories"`
	TopActions  []string        `json:"top_actions"`
}

// ComputeScore вычисляет рейтинг на основе analysis + policy + certs.
func ComputeScore(analysis AnalysisResult, policyCount, certHigh int) ScoreReport {
	cats := []ScoreCategory{
		{Name: "security", Weight: 0.30, Score: 100},
		{Name: "reliability", Weight: 0.25, Score: 100},
		{Name: "performance", Weight: 0.20, Score: 100},
		{Name: "maintainability", Weight: 0.15, Score: 100},
		{Name: "observability", Weight: 0.10, Score: 100},
	}

	penalty := func(sev Severity) float64 {
		switch sev {
		case SeverityHigh:
			return 15
		case SeverityMedium:
			return 8
		default:
			return 3
		}
	}

	issues := CollectIssues(analysis)
	var actions []string
	for _, iss := range issues {
		p := penalty(iss.Severity)
		switch iss.Type {
		case "ssl_missing", "ssl_protocols_weak", "ssl_ciphers_weak", "missing_security_header", "autoindex_on":
			cats[0].Score -= p
			cats[0].Issues++
		case "listen_servername_conflict", "location_conflict", "rewrite_cycle":
			cats[1].Score -= p
			cats[1].Issues++
		case "limit_too_small", "limit_too_large", "no_limit_req_conn":
			cats[2].Score -= p
			cats[2].Issues++
		case "duplicate_directive", "empty_block", "dead_location", "unused_variable":
			cats[3].Score -= p
			cats[3].Issues++
		default:
			cats[3].Score -= p / 2
		}
		if iss.Severity == SeverityHigh && len(actions) < 3 {
			actions = append(actions, iss.Type+": "+iss.Solution)
		}
	}
	cats[0].Score -= float64(certHigh) * 20
	cats[0].Score -= float64(policyCount) * 2

	for i := range cats {
		if cats[i].Score < 0 {
			cats[i].Score = 0
		}
		if cats[i].Score > 100 {
			cats[i].Score = 100
		}
	}

	var total float64
	for _, c := range cats {
		total += c.Score * c.Weight
	}
	return ScoreReport{Total: total, Categories: cats, TopActions: actions}
}
