package hub

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/version"
	"github.com/rainysundaynight/nginx-lens/internal/webauth"
)

// ---------- HTTP hub server ----------
// Агрегация snapshot с агентов и dashboard UI.

// NewRouter создаёт HTTP router для hub.
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware())

	staticSub, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": version.Version})
	})
	r.Get("/", handleDashboard)

	r.Group(func(r chi.Router) {
		r.Use(webauth.VerifyHubToken)
		r.Get("/api/v1/agents", handleAgents)
		r.Get("/api/v1/snapshots", handleSnapshots)
		r.Get("/api/v1/status", handleStatus)
		r.Get("/api/v1/explain", handleExplain)
		r.Get("/api/v1/hub/state", handleHubState)
		r.Get("/api/agents", handleAgents)
		r.Get("/api/snapshots", handleSnapshots)
		r.Get("/api/status", handleStatus)
	})

	return r
}

func corsMiddleware() func(http.Handler) http.Handler {
	origins := parseCORSOrigins()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := "*"
			if len(origins) > 0 && origins[0] != "*" {
				origin = origins[0]
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			next.ServeHTTP(w, r)
		})
	}
}

func handleDashboard(w http.ResponseWriter, _ *http.Request) {
	data, err := templatesFS.ReadFile("templates/dashboard.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	html := string(data)
	refresh := config.Get().Config.Web.Hub.RefreshInterval
	if refresh <= 0 {
		refresh = 30
	}
	html = strings.ReplaceAll(html, "{{ refresh_interval }}", strconv.Itoa(refresh))
	html = strings.ReplaceAll(html, "{{ version }}", version.Version)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func handleHubState(w http.ResponseWriter, _ *http.Request) {
	agents := parseAgents()
	results := fetchSnapshots(agents)
	refresh := config.Get().Config.Web.Hub.RefreshInterval
	if refresh <= 0 {
		refresh = 30
	}
	json.NewEncoder(w).Encode(BuildHubState(results, version.Version, refresh))
}

func handleAgents(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string][]string{"agents": parseAgents()})
}

func handleSnapshots(w http.ResponseWriter, _ *http.Request) {
	agents := parseAgents()
	results := fetchSnapshots(agents)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents":  agents,
		"results": results,
	})
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	agents := parseAgents()
	results := fetchSnapshots(agents)
	var statuses []map[string]interface{}
	online := 0
	for _, r := range results {
		if r["online"].(bool) {
			online++
		}
		statuses = append(statuses, map[string]interface{}{
			"agent":  r["agent"],
			"online": r["online"],
			"error":  r["error"],
		})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents_total":  len(agents),
		"agents_online": online,
		"statuses":      statuses,
	})
}

func parseAgents() []string {
	if env := os.Getenv("NGINX_LENS_AGENTS"); env != "" {
		var agents []string
		for _, a := range strings.Split(env, ",") {
			if s := strings.TrimSpace(a); s != "" {
				agents = append(agents, s)
			}
		}
		return agents
	}
	return config.Get().Config.Web.Hub.Agents
}

func parseCORSOrigins() []string {
	if env := os.Getenv("NGINX_LENS_CORS_ORIGINS"); env != "" {
		if env == "*" {
			return []string{"*"}
		}
		return strings.Split(env, ",")
	}
	return config.Get().Config.Web.Hub.CORSOrigins
}

func fetchSnapshots(agents []string) []map[string]interface{} {
	var results []map[string]interface{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	headers := webauth.AgentHeaders()
	client := &http.Client{}

	for _, agentURL := range agents {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			snapURL := strings.TrimRight(url, "/") + "/snapshot"
			req, _ := http.NewRequest(http.MethodGet, snapURL, nil)
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			item := map[string]interface{}{"agent": url, "online": false}
			resp, err := client.Do(req)
			if err != nil {
				item["error"] = err.Error()
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var snap map[string]interface{}
					if json.NewDecoder(resp.Body).Decode(&snap) == nil {
						item["online"] = true
						item["snapshot"] = snap
					}
				} else {
					item["error"] = resp.Status
				}
			}
			mu.Lock()
			results = append(results, item)
			mu.Unlock()
		}(agentURL)
	}
	wg.Wait()
	return results
}
