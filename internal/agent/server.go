package agent

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rainysundaynight/nginx-lens/internal/version"
	"github.com/rainysundaynight/nginx-lens/internal/webauth"
)

// ---------- HTTP agent server ----------
// /healthz, /version, /snapshot, /explain, /metrics.

// NewRouter создаёт HTTP router для agent.
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": version.Version})
	})
	r.Get("/metrics", handleMetrics)
	r.Group(func(r chi.Router) {
		r.Use(webauth.VerifyAgentToken)
		r.Get("/snapshot", handleSnapshot)
		r.Get("/explain", handleExplain)
	})

	return r
}

func handleSnapshot(w http.ResponseWriter, _ *http.Request) {
	snap, err := CollectSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hostname, _ := os.Hostname()
	resp := map[string]interface{}{
		"config_path":       snap.ConfigPath,
		"cache":             snap.Cache,
		"dynamic_upstream":  snap.DynamicUpstream,
		"analyze":           snap.Analyze,
		"upstreams":         snap.Upstreams,
		"upstreams_health":  snap.UpstreamsHealth,
		"dependency_graph":  snap.DependencyGraph,
		"stream_graph":      snap.StreamGraph,
		"score":             snap.Score,
		"policy_issues":     snap.PolicyIssues,
		"cert_issues":       snap.CertIssues,
		"cert_timeline":     snap.CertTimeline,
		"error_stats":       snap.ErrorStats,
		"access_stats":      snap.AccessStats,
		"log_correlations":  snap.LogCorrelations,
		"nginx_build":       snap.NginxBuild,
		"module_groups":     snap.ModuleGroups,
		"module_issues":     snap.ModuleIssues,
		"meta": map[string]interface{}{
			"hostname":      hostname,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
			"agent_version": version.Version,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
