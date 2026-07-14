package hub

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
	// Публичный статус auth: нужен ли hub token для API (без раскрытия секрета).
	r.Get("/api/v1/auth", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"required": webauth.HubToken() != ""})
	})
	r.Get("/", handleDashboard)

	r.Group(func(r chi.Router) {
		r.Use(webauth.VerifyHubToken)
		r.Get("/api/v1/agents", handleAgents)
		r.Post("/api/v1/agents", handleAddAgent)
		r.Delete("/api/v1/agents", handleDeleteAgent)
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
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
	html = strings.ReplaceAll(html, "{{ auth_required }}", strconv.FormatBool(webauth.HubToken() != ""))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func handleHubState(w http.ResponseWriter, _ *http.Request) {
	agents := listAgents()
	results := fetchSnapshots(agents)
	refresh := config.Get().Config.Web.Hub.RefreshInterval
	if refresh <= 0 {
		refresh = 30
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BuildHubState(results, version.Version, refresh))
}

func handleAgents(w http.ResponseWriter, _ *http.Request) {
	items := listAgents()
	urls := make([]string, 0, len(items))
	for _, a := range items {
		urls = append(urls, a.URL)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": urls,
		"items":  items,
	})
}

// handleAddAgent — POST /api/v1/agents {url, region?, name?}.
func handleAddAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL    string `json:"url"`
		Region string `json:"region"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ep, err := addRuntimeAgent(body.URL, body.Region, body.Name)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errAgentExists) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ep)
}

// handleDeleteAgent — DELETE /api/v1/agents?url=.
func handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if err := removeRuntimeAgent(rawURL); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errAgentNotRuntime) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func handleSnapshots(w http.ResponseWriter, _ *http.Request) {
	agents := listAgents()
	results := fetchSnapshots(agents)
	urls := make([]string, 0, len(agents))
	for _, a := range agents {
		urls = append(urls, a.URL)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents":  urls,
		"results": results,
	})
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	agents := listAgents()
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents_total":  len(agents),
		"agents_online": online,
		"statuses":      statuses,
	})
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

// fetchSnapshots параллельно тянет /snapshot с агентов и меряет scrape latency.
func fetchSnapshots(agents []AgentEndpoint) []map[string]interface{} {
	var results []map[string]interface{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	headers := webauth.AgentHeaders()
	client := &http.Client{Timeout: 15 * time.Second}

	for _, ep := range agents {
		wg.Add(1)
		go func(agent AgentEndpoint) {
			defer wg.Done()
			snapURL := strings.TrimRight(agent.URL, "/") + "/snapshot"
			req, _ := http.NewRequest(http.MethodGet, snapURL, nil)
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			item := map[string]interface{}{
				"agent":  agent.URL,
				"online": false,
				"region": agent.Region,
			}
			if agent.Name != "" {
				item["label"] = agent.Name
			}
			start := time.Now()
			resp, err := client.Do(req)
			item["scrape_ms"] = float64(time.Since(start).Milliseconds())
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
		}(ep)
	}
	wg.Wait()
	return results
}
