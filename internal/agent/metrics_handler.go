package agent

import (
	"fmt"
	"net/http"
	"strings"
)

// handleMetrics отдаёт Prometheus OpenMetrics с метриками snapshot.
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	snap, err := CollectSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var b strings.Builder
	writeGauge := func(name, help string, val float64) {
		b.WriteString(fmt.Sprintf("# HELP %s %s\n# TYPE %s gauge\n%s %.2f\n", name, help, name, name, val))
	}
	writeLabeled := func(name, help string, labels map[string]string, val float64) {
		b.WriteString(fmt.Sprintf("# HELP %s %s\n# TYPE %s gauge\n", name, help, name))
		var parts []string
		for k, v := range labels {
			parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
		}
		b.WriteString(fmt.Sprintf("%s{%s} %.2f\n", name, strings.Join(parts, ","), val))
	}

	writeGauge("nginx_lens_score", "Config score 0-100", snap.Score.Total)
	upOk, upTotal := 0, 0
	for name, rows := range snap.UpstreamsHealth {
		for _, row := range rows {
			upTotal++
			val := 0.0
			if row.Healthy {
				upOk++
				val = 1
			}
			writeLabeled("nginx_lens_upstream_healthy", "Upstream server health 0/1", map[string]string{
				"upstream": name,
				"address":  row.Address,
			}, val)
		}
	}
	if upTotal > 0 {
		writeGauge("nginx_lens_upstream_healthy_ratio", "Healthy upstream ratio", float64(upOk)/float64(upTotal))
		writeGauge("nginx_lens_upstream_servers_total", "Total upstream servers", float64(upTotal))
	}
	minDays := 365.0
	for _, c := range snap.CertIssues {
		if c.DaysLeft > 0 && float64(c.DaysLeft) < minDays {
			minDays = float64(c.DaysLeft)
		}
	}
	if len(snap.CertIssues) > 0 {
		writeGauge("nginx_lens_cert_days_left_min", "Min cert days left", minDays)
	}
	if snap.AccessStats != nil {
		writeGauge("nginx_lens_access_rps", "Access log RPS", snap.AccessStats.RPS)
		writeGauge("nginx_lens_access_5xx_total", "Access 5xx count", float64(snap.AccessStats.Status5xx))
	}
	writeGauge("nginx_lens_issues_high", "High severity issues", float64(snap.Analyze.Summary["high"]))
	for _, c := range snap.LogCorrelations {
		if c.ErrorTotal > 0 {
			writeLabeled("nginx_lens_upstream_errors_total", "Correlated error log count", map[string]string{
				"upstream": c.Upstream,
			}, float64(c.ErrorTotal))
		}
	}
	w.Write([]byte(b.String()))
}
