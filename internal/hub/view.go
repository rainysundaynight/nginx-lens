package hub

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/rainysundaynight/nginx-lens/internal/logs"
)

// ---------- View-модели Hub UI ----------
// Нормализация raw agent snapshot для reboot-дизайна.

// HubState — агрегированное состояние для SPA.
type HubState struct {
	Meta         HubMeta          `json:"meta"`
	KPI          HubKPI           `json:"kpi"`
	HealthBars   []HubHealthBar   `json:"health_bars"`
	Severity     HubSeverity      `json:"severity"`
	Snapshots    []HubSnapshot    `json:"snapshots"`
	Correlations []HubCorrelation `json:"correlations"`
	BlastRadius  []HubBlastGroup  `json:"blast_radius"`
}

// HubMeta — метаданные hub.
type HubMeta struct {
	Version         string `json:"version"`
	RefreshInterval int    `json:"refresh_interval"`
	UpdatedAt       string `json:"updated_at"`
	SystemStatus    string `json:"system_status"`
	AgentsOnline    int    `json:"agents_online"`
	AgentsTotal     int    `json:"agents_total"`
}

// HubKPI — карточки KPI.
type HubKPI struct {
	AgentsOnline    string `json:"agents_online"`
	AgentsSuffix    string `json:"agents_suffix"`
	CriticalIssues  string `json:"critical_issues"`
	Warnings        string `json:"warnings"`
	UpstreamHealthy string `json:"upstream_healthy"`
}

// HubHealthBar — столбец health overview.
type HubHealthBar struct {
	Agent string  `json:"agent"`
	Pct   float64 `json:"pct"`
	Bad   bool    `json:"bad"`
}

// HubSeverity — breakdown по severity.
type HubSeverity struct {
	High      int `json:"high"`
	Medium    int `json:"medium"`
	Low       int `json:"low"`
	HighPct   int `json:"high_pct"`
	MediumPct int `json:"medium_pct"`
	LowPct    int `json:"low_pct"`
}

// HubSnapshot — карточка/деталь агента.
type HubSnapshot struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	URL             string             `json:"url"`
	Host            string             `json:"host"`
	Status          string             `json:"status"`
	ConfigScore     int                `json:"config_score"`
	Version         string             `json:"version"`
	UpdatedAt       string             `json:"updated_at"`
	Categories      HubCategories      `json:"categories"`
	IssuesBreakdown map[string]int     `json:"issues_breakdown"`
	Severity        HubSeverityShort   `json:"severity"`
	Modules         []string           `json:"modules"`
	Note            string             `json:"note,omitempty"`
	Error           string             `json:"error,omitempty"`
	Upstreams       []HubUpstreamRow   `json:"upstreams"`
	Build           []HubKV            `json:"build"`
	Issues          []HubIssueRow      `json:"issues"`
	Certs           []HubCertRow       `json:"certs"`
	Blast           []HubBlastRow      `json:"blast"`
	Errors          []HubErrorRow      `json:"errors"`
	Access          *HubAccessMini     `json:"access,omitempty"`
}

// HubCategories — score breakdown.
type HubCategories struct {
	Security        float64 `json:"security"`
	Reliability     float64 `json:"reliability"`
	Performance     float64 `json:"performance"`
	Maintainability float64 `json:"maintainability"`
	Observability   float64 `json:"observability"`
}

// HubSeverityShort — high/med/low для карточки.
type HubSeverityShort struct {
	High int `json:"high"`
	Med  int `json:"med"`
	Low  int `json:"low"`
}

// HubUpstreamRow — строка upstream tab.
type HubUpstreamRow struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
	Errors  string `json:"errors"`
	Stream  bool   `json:"stream,omitempty"`
}

// HubKV — key-value для build tab.
type HubKV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HubIssueRow — issue в списке.
type HubIssueRow struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Rule     string `json:"rule"`
	Location string `json:"location"`
}

// HubCertRow — сертификат.
type HubCertRow struct {
	Domain   string `json:"domain"`
	Issuer   string `json:"issuer"`
	Expires  string `json:"expires"`
	DaysLeft int    `json:"days_left"`
}

// HubBlastRow — blast entry.
type HubBlastRow struct {
	Upstream string `json:"upstream"`
	Location string `json:"location"`
	Impact   int    `json:"impact"`
}

// HubErrorRow — error log строка.
type HubErrorRow struct {
	Time    string `json:"time"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HubAccessMini — access stats в карточке.
type HubAccessMini struct {
	RPS       float64 `json:"rps"`
	Status5xx int     `json:"status_5xx"`
	P95Ms     float64 `json:"p95_ms"`
}

// HubCorrelation — correlation page item.
type HubCorrelation struct {
	ID        string   `json:"id"`
	Time      string   `json:"time"`
	Upstream  string   `json:"upstream"`
	Error     string   `json:"error"`
	Locations []string `json:"locations"`
	Matches   int      `json:"matches"`
	Severity  string   `json:"severity"`
	Agent     string   `json:"agent"`
}

// HubBlastGroup — blast-radius page group.
type HubBlastGroup struct {
	Upstream  string        `json:"upstream"`
	Health    string        `json:"health"`
	Agent     string        `json:"agent"`
	Locations []HubBlastLoc `json:"locations"`
}

// HubBlastLoc — location в blast group.
type HubBlastLoc struct {
	Loc      string `json:"loc"`
	Impact   int    `json:"impact"`
	Requests string `json:"requests"`
}

// BuildHubState собирает view из raw agent results.
func BuildHubState(results []map[string]interface{}, hubVersion string, refreshSec int) HubState {
	now := time.Now().UTC().Format(time.RFC3339)
	state := HubState{
		Meta: HubMeta{
			Version:         hubVersion,
			RefreshInterval: refreshSec,
			UpdatedAt:       now,
		},
	}
	var totalHigh, totalMed, totalLow int
	upOk, upTotal := 0, 0

	for _, item := range results {
		snap := buildHubSnapshot(item)
		state.Snapshots = append(state.Snapshots, snap)
		if snap.Status == "online" || snap.Status == "warning" {
			state.Meta.AgentsOnline++
			totalHigh += snap.Severity.High
			totalMed += snap.Severity.Med
			totalLow += snap.Severity.Low
			for _, u := range snap.Upstreams {
				upTotal++
				if u.Status == "OK" {
					upOk++
				}
			}
			pct := upstreamPctFromSnapshot(item)
			state.HealthBars = append(state.HealthBars, HubHealthBar{
				Agent: snap.Name,
				Pct:   pct,
				Bad:   pct < 100 && pct >= 0,
			})
		}
		state.Correlations = append(state.Correlations, buildCorrelations(item, snap)...)
		state.BlastRadius = append(state.BlastRadius, buildBlastGroups(item, snap)...)
	}

	state.Meta.AgentsTotal = len(results)
	state.Meta.SystemStatus = systemStatus(state.Meta.AgentsOnline, state.Meta.AgentsTotal, totalHigh)

	totalIssues := totalHigh + totalMed + totalLow
	state.Severity = HubSeverity{High: totalHigh, Medium: totalMed, Low: totalLow}
	if totalIssues > 0 {
		state.Severity.HighPct = int(math.Round(float64(totalHigh) / float64(totalIssues) * 100))
		state.Severity.MediumPct = int(math.Round(float64(totalMed) / float64(totalIssues) * 100))
		state.Severity.LowPct = int(math.Round(float64(totalLow) / float64(totalIssues) * 100))
	}

	upPct := "—"
	if upTotal > 0 {
		upPct = fmt.Sprintf("%.1f%%", float64(upOk)/float64(upTotal)*100)
	}
	state.KPI = HubKPI{
		AgentsOnline:    fmt.Sprintf("%d", state.Meta.AgentsOnline),
		AgentsSuffix:    fmt.Sprintf("/ %d", state.Meta.AgentsTotal),
		CriticalIssues:  fmt.Sprintf("%02d", totalHigh),
		Warnings:        fmt.Sprintf("%02d", totalMed),
		UpstreamHealthy: upPct,
	}
	return state
}

func systemStatus(online, total, high int) string {
	if total == 0 || online == 0 {
		return "OFFLINE"
	}
	if online < total || high > 0 {
		return "DEGRADED"
	}
	return "NOMINAL"
}

func buildHubSnapshot(item map[string]interface{}) HubSnapshot {
	agentURL, _ := item["agent"].(string)
	online, _ := item["online"].(bool)
	errMsg, _ := item["error"].(string)
	raw, _ := item["snapshot"].(map[string]interface{})

	id := agentSlug(agentURL)
	name := id
	host := agentHost(agentURL)
	status := "offline"
	if online {
		status = "online"
		if raw != nil {
			if sev := mapInt(raw, "analyze", "summary", "high"); sev > 0 {
				status = "warning"
			}
		}
	}

	s := HubSnapshot{
		ID: id, Name: name, URL: agentURL, Host: host,
		Status: status, Error: errMsg,
		UpdatedAt: time.Now().Format("02.01.2006, 15:04:05"),
	}
	if !online || raw == nil {
		return s
	}

	meta, _ := raw["meta"].(map[string]interface{})
	if h, ok := meta["hostname"].(string); ok && h != "" {
		s.Name = h
	}

	if score, ok := raw["score"].(map[string]interface{}); ok {
		if v, ok := score["total"].(float64); ok {
			s.ConfigScore = int(math.Round(v))
		}
		s.Categories = parseCategories(score)
	}

	analyze, _ := raw["analyze"].(map[string]interface{})
	summary, _ := analyze["issues"].([]interface{})
	s.Severity = HubSeverityShort{
		High: intMapVal(analyze, "summary", "high"),
		Med:  intMapVal(analyze, "summary", "medium"),
		Low:  intMapVal(analyze, "summary", "low"),
	}
	if score, ok := raw["score"].(map[string]interface{}); ok {
		s.IssuesBreakdown = parseCategoryIssues(score)
	}
	s.Issues = parseIssues(summary)

	build, _ := raw["nginx_build"].(map[string]interface{})
	if build != nil {
		if v, ok := build["version"].(string); ok {
			s.Version = v
		}
		s.Modules = parseModules(build, raw)
		s.Build = parseBuild(build)
	}

	s.Upstreams = parseUpstreams(raw)
	s.Certs = parseCerts(raw)
	s.Blast = parseBlast(raw)
	s.Errors = parseErrors(raw)

	if access, ok := raw["access_stats"].(map[string]interface{}); ok {
		s.Access = &HubAccessMini{
			RPS:       floatVal(access["rps"]),
			Status5xx: int(floatVal(access["status_5xx"])),
			P95Ms:     floatVal(access["p95_latency_ms"]),
		}
	} else {
		s.Note = "access.log не настроен"
	}

	return s
}

func agentSlug(agentURL string) string {
	u, err := url.Parse(agentURL)
	if err != nil || u.Host == "" {
		return strings.TrimPrefix(strings.TrimPrefix(agentURL, "http://"), "https://")
	}
	return strings.ReplaceAll(u.Host, ":", "-")
}

func agentHost(agentURL string) string {
	u, err := url.Parse(agentURL)
	if err != nil || u.Host == "" {
		return agentURL
	}
	return u.Host
}

func parseCategories(score map[string]interface{}) HubCategories {
	cats, _ := score["categories"].([]interface{})
	out := HubCategories{Security: 100, Reliability: 100, Performance: 100, Maintainability: 100, Observability: 100}
	for _, c := range cats {
		m, _ := c.(map[string]interface{})
		if m == nil {
			continue
		}
		name, _ := m["name"].(string)
		val := floatVal(m["score"])
		switch name {
		case "security":
			out.Security = val
		case "reliability":
			out.Reliability = val
		case "performance":
			out.Performance = val
		case "maintainability":
			out.Maintainability = val
		case "observability":
			out.Observability = val
		}
	}
	return out
}

func parseCategoryIssues(score map[string]interface{}) map[string]int {
	out := map[string]int{
		"security": 0, "reliability": 0, "performance": 0,
		"maintainability": 0, "observability": 0,
	}
	cats, _ := score["categories"].([]interface{})
	for _, c := range cats {
		m, _ := c.(map[string]interface{})
		if m == nil {
			continue
		}
		name, _ := m["name"].(string)
		if _, ok := out[name]; !ok {
			continue
		}
		out[name] = int(floatVal(m["issues"]))
	}
	return out
}

func parseIssues(raw []interface{}) []HubIssueRow {
	var out []HubIssueRow
	for i, it := range raw {
		m, _ := it.(map[string]interface{})
		if m == nil {
			continue
		}
		sev, _ := m["severity"].(string)
		if sev == "medium" {
			sev = "med"
		}
		loc := stringVal(m["file"])
		if line := int(floatVal(m["line"])); line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, line)
		}
		out = append(out, HubIssueRow{
			ID:       fmt.Sprintf("I-%03d", i+1),
			Severity: sev,
			Title:    stringVal(m["description"]),
			Rule:     stringVal(m["type"]),
			Location: loc,
		})
	}
	return out
}

func parseModules(build map[string]interface{}, raw map[string]interface{}) []string {
	var mods []string
	if v, ok := build["version"].(string); ok && v != "" {
		mods = append(mods, "nginx/"+v)
	}
	if groups, ok := raw["module_groups"].(map[string]interface{}); ok {
		for _, key := range []string{"core", "ssl", "stream", "third_party"} {
			arr, _ := groups[key].([]interface{})
			for _, m := range arr {
				if s, ok := m.(string); ok {
					mods = append(mods, s)
				}
			}
		}
	}
	if len(mods) <= 1 {
		for _, key := range []string{"static_modules", "dynamic_modules"} {
			arr, _ := build[key].([]interface{})
			for _, m := range arr {
				if s, ok := m.(string); ok {
					mods = append(mods, s)
				}
			}
		}
	}
	return mods
}

func parseBuild(build map[string]interface{}) []HubKV {
	var rows []HubKV
	if v := stringVal(build["built_by"]); v != "" {
		rows = append(rows, HubKV{Name: "Built by", Value: v})
	}
	if v := stringVal(build["openssl"]); v != "" {
		rows = append(rows, HubKV{Name: "OpenSSL", Value: v})
	}
	if v := stringVal(build["tls"]); v != "" {
		rows = append(rows, HubKV{Name: "TLS", Value: v})
	}
	if v := stringVal(build["configure_args"]); v != "" {
		if len(v) > 120 {
			v = v[:120] + "…"
		}
		rows = append(rows, HubKV{Name: "Configure", Value: v})
	}
	return rows
}

func parseUpstreams(raw map[string]interface{}) []HubUpstreamRow {
	health, _ := raw["upstreams_health"].(map[string]interface{})
	stream, _ := raw["stream_graph"].(map[string]interface{})
	streamNames := map[string]struct{}{}
	for name := range stream {
		streamNames[name] = struct{}{}
	}
	var rows []HubUpstreamRow
	for name, val := range health {
		servers, _ := val.([]interface{})
		for _, srv := range servers {
			m, _ := srv.(map[string]interface{})
			if m == nil {
				continue
			}
			st := "—"
			if h, ok := m["healthy"].(bool); ok {
				if h {
					st = "OK"
				} else {
					st = "DOWN"
				}
			}
			errN := "—"
			if cors, ok := raw["log_correlations"].([]interface{}); ok {
				for _, c := range cors {
					cm, _ := c.(map[string]interface{})
					if cm == nil {
						continue
					}
					if stringVal(cm["upstream"]) == name {
						errN = fmt.Sprintf("%d", int(floatVal(cm["error_total"])))
					}
				}
			}
			_, isStream := streamNames[name]
			rows = append(rows, HubUpstreamRow{
				Name: name, Address: stringVal(m["address"]),
				Status: st, Errors: errN, Stream: isStream,
			})
		}
	}
	return rows
}

func parseCerts(raw map[string]interface{}) []HubCertRow {
	timeline, _ := raw["cert_timeline"].([]interface{})
	var rows []HubCertRow
	for _, t := range timeline {
		m, _ := t.(map[string]interface{})
		if m == nil {
			continue
		}
		exp := ""
		if ts, ok := m["expires_at"].(string); ok && len(ts) >= 10 {
			exp = ts[:10]
		}
		rows = append(rows, HubCertRow{
			Domain:   stringVal(m["server_name"]),
			Issuer:   "—",
			Expires:  exp,
			DaysLeft: int(floatVal(m["days_left"])),
		})
	}
	return rows
}

func parseBlast(raw map[string]interface{}) []HubBlastRow {
	var rows []HubBlastRow
	graph, _ := raw["dependency_graph"].(map[string]interface{})
	for up, entries := range graph {
		arr, _ := entries.([]interface{})
		for _, e := range arr {
			m, _ := e.(map[string]interface{})
			if m == nil {
				continue
			}
			impact := 0
			if cors, ok := raw["log_correlations"].([]interface{}); ok {
				for _, c := range cors {
					cm, _ := c.(map[string]interface{})
					if cm != nil && stringVal(cm["upstream"]) == up {
						impact = int(floatVal(cm["access_5xx_pct"]))
					}
				}
			}
			rows = append(rows, HubBlastRow{
				Upstream: up,
				Location: stringVal(m["location"]),
				Impact:   impact,
			})
		}
	}
	return rows
}

func parseErrors(raw map[string]interface{}) []HubErrorRow {
	es, _ := raw["error_stats"].(map[string]interface{})
	if es == nil {
		return nil
	}
	index := logs.BuildUpstreamIndex(rawUpstreams(raw))
	arr, _ := es["upstream_errors"].([]interface{})
	var rows []HubErrorRow
	for _, e := range arr {
		m, _ := e.(map[string]interface{})
		if m == nil {
			continue
		}
		upRaw := stringVal(m["upstream"])
		upName := logs.ResolveLogicalUpstream(upRaw, index)
		if upName == "" {
			upName = upRaw
		}
		msg := stringVal(m["message"])
		if msg == "" || msg == "upstream errors" {
			msg = fmt.Sprintf("%d upstream error(s)", int(floatVal(m["count"])))
		}
		rows = append(rows, HubErrorRow{
			Time:    time.Now().Format("15:04:05"),
			Code:    "ERR",
			Message: fmt.Sprintf("%s · %s", upName, msg),
		})
	}
	return rows
}

func rawUpstreams(raw map[string]interface{}) map[string][]string {
	src, _ := raw["upstreams"].(map[string]interface{})
	out := make(map[string][]string, len(src))
	for name, val := range src {
		arr, _ := val.([]interface{})
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out[name] = append(out[name], s)
			}
		}
	}
	return out
}

func buildCorrelations(item map[string]interface{}, snap HubSnapshot) []HubCorrelation {
	raw, _ := item["snapshot"].(map[string]interface{})
	if raw == nil {
		return nil
	}
	cors, _ := raw["log_correlations"].([]interface{})
	var out []HubCorrelation
	for i, c := range cors {
		m, _ := c.(map[string]interface{})
		if m == nil {
			continue
		}
		locs := []string{}
		if arr, ok := m["locations"].([]interface{}); ok {
			for _, l := range arr {
				if s, ok := l.(string); ok {
					locs = append(locs, s)
				}
			}
		}
		sev := "med"
		if floatVal(m["access_5xx_pct"]) > 10 || floatVal(m["error_total"]) > 50 {
			sev = "high"
		}
		out = append(out, HubCorrelation{
			ID:       fmt.Sprintf("C-%04d", i+1),
			Time:     time.Now().Format("2006-01-02 15:04:05"),
			Upstream: stringVal(m["upstream"]),
			Error: fmt.Sprintf("access 5xx %.1f%%, errors %d, connect %d",
				floatVal(m["access_5xx_pct"]), int(floatVal(m["error_total"])), int(floatVal(m["error_connect_failed"]))),
			Locations: locs,
			Matches:   int(floatVal(m["error_total"])) + int(floatVal(m["access_502"])),
			Severity:  sev,
			Agent:     snap.Name,
		})
	}
	return out
}

func buildBlastGroups(item map[string]interface{}, snap HubSnapshot) []HubBlastGroup {
	raw, _ := item["snapshot"].(map[string]interface{})
	if raw == nil {
		return nil
	}
	graph, _ := raw["dependency_graph"].(map[string]interface{})
	health, _ := raw["upstreams_health"].(map[string]interface{})
	var groups []HubBlastGroup
	for up, entries := range graph {
		h := "healthy"
		if rows, ok := health[up].([]interface{}); ok {
			for _, r := range rows {
				m, _ := r.(map[string]interface{})
				if m != nil && m["healthy"] == false {
					h = "critical"
				}
			}
		}
		if h == "healthy" {
			for _, b := range snap.Blast {
				if b.Upstream == up && b.Impact >= 20 {
					h = "degraded"
				}
			}
		}
		var locs []HubBlastLoc
		arr, _ := entries.([]interface{})
		for _, e := range arr {
			m, _ := e.(map[string]interface{})
			if m == nil {
				continue
			}
			impact := 0
			for _, b := range snap.Blast {
				if b.Location == stringVal(m["location"]) {
					impact = b.Impact
				}
			}
			reqs := "—"
			if snap.Access != nil && snap.Access.RPS > 0 {
				reqs = fmt.Sprintf("%.1f rps", snap.Access.RPS)
			}
			locs = append(locs, HubBlastLoc{
				Loc: stringVal(m["location"]), Impact: impact, Requests: reqs,
			})
		}
		groups = append(groups, HubBlastGroup{
			Upstream: up, Health: h, Agent: snap.Name, Locations: locs,
		})
	}
	return groups
}

func upstreamPctFromSnapshot(item map[string]interface{}) float64 {
	raw, _ := item["snapshot"].(map[string]interface{})
	if raw == nil {
		return -1
	}
	health, _ := raw["upstreams_health"].(map[string]interface{})
	ok, total := 0, 0
	for _, val := range health {
		servers, _ := val.([]interface{})
		for _, srv := range servers {
			total++
			m, _ := srv.(map[string]interface{})
			if m != nil && m["healthy"] == true {
				ok++
			}
		}
	}
	if total == 0 {
		return 100
	}
	return float64(ok) / float64(total) * 100
}

func mapInt(raw map[string]interface{}, keys ...string) int {
	m := raw
	for i, k := range keys {
		if i == len(keys)-1 {
			return intMapVal(m, k)
		}
		next, _ := m[k].(map[string]interface{})
		if next == nil {
			return 0
		}
		m = next
	}
	return 0
}

func intMapVal(m map[string]interface{}, keys ...string) int {
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			if v, ok := cur[k].(float64); ok {
				return int(v)
			}
			return 0
		}
		next, _ := cur[k].(map[string]interface{})
		if next == nil {
			return 0
		}
		cur = next
	}
	return 0
}

func floatVal(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

func stringVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
