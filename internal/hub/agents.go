package hub

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rainysundaynight/nginx-lens/internal/config"
)

// ---------- Runtime agent registry ----------
// Динамические агенты (Add agent) поверх YAML/env, с persist в файл.

// AgentEndpoint — нормализованная запись агента для fan-out.
type AgentEndpoint struct {
	URL    string `json:"url"`
	Region string `json:"region,omitempty"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source"` // config | env | runtime
}

var (
	runtimeMu      sync.RWMutex
	runtimeAgents  []AgentEndpoint
	runtimeLoaded  bool
)

// listAgents объединяет config/env и runtime-агентов (dedupe по URL).
func listAgents() []AgentEndpoint {
	loadRuntimeAgents()
	base := parseConfiguredAgents()
	runtimeMu.RLock()
	extra := append([]AgentEndpoint(nil), runtimeAgents...)
	runtimeMu.RUnlock()

	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]AgentEndpoint, 0, len(base)+len(extra))
	for _, a := range append(base, extra...) {
		key := normalizeAgentURL(a.URL)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		a.URL = key
		out = append(out, a)
	}
	return out
}

// parseConfiguredAgents читает агентов из NGINX_LENS_AGENTS или YAML.
func parseConfiguredAgents() []AgentEndpoint {
	if env := os.Getenv("NGINX_LENS_AGENTS"); env != "" {
		var out []AgentEndpoint
		for _, part := range strings.Split(env, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// url@region или url|region|name
			ep := AgentEndpoint{URL: part, Source: "env"}
			if strings.Contains(part, "|") {
				bits := strings.Split(part, "|")
				ep.URL = strings.TrimSpace(bits[0])
				if len(bits) > 1 {
					ep.Region = strings.TrimSpace(bits[1])
				}
				if len(bits) > 2 {
					ep.Name = strings.TrimSpace(bits[2])
				}
			} else if at := strings.LastIndex(part, "@"); at > 0 {
				ep.URL = part[:at]
				ep.Region = part[at+1:]
			}
			out = append(out, ep)
		}
		return out
	}
	var out []AgentEndpoint
	for _, a := range config.Get().Config.Web.Hub.Agents {
		out = append(out, AgentEndpoint{
			URL: a.URL, Region: a.Region, Name: a.Name, Source: "config",
		})
	}
	return out
}

func normalizeAgentURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

func runtimeAgentsPath() string {
	if p := os.Getenv("NGINX_LENS_HUB_AGENTS_FILE"); p != "" {
		return p
	}
	dir := filepath.Dir(config.Get().ConfigPath)
	if dir == "" || dir == "." {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "hub-agents.runtime.json")
}

func loadRuntimeAgents() {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if runtimeLoaded {
		return
	}
	runtimeLoaded = true
	data, err := os.ReadFile(runtimeAgentsPath())
	if err != nil {
		return
	}
	var items []AgentEndpoint
	if json.Unmarshal(data, &items) != nil {
		return
	}
	for i := range items {
		items[i].URL = normalizeAgentURL(items[i].URL)
		items[i].Source = "runtime"
	}
	runtimeAgents = items
}

func saveRuntimeAgentsLocked() error {
	path := runtimeAgentsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.MarshalIndent(runtimeAgents, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// addRuntimeAgent регистрирует агента и сохраняет на диск.
func addRuntimeAgent(url, region, name string) (AgentEndpoint, error) {
	url = normalizeAgentURL(url)
	if url == "" {
		return AgentEndpoint{}, errInvalidAgentURL
	}
	loadRuntimeAgents()
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	for _, a := range listAgentsUnlocked() {
		if normalizeAgentURL(a.URL) == url {
			return AgentEndpoint{}, errAgentExists
		}
	}
	ep := AgentEndpoint{URL: url, Region: region, Name: name, Source: "runtime"}
	runtimeAgents = append(runtimeAgents, ep)
	if err := saveRuntimeAgentsLocked(); err != nil {
		runtimeAgents = runtimeAgents[:len(runtimeAgents)-1]
		return AgentEndpoint{}, err
	}
	return ep, nil
}

func listAgentsUnlocked() []AgentEndpoint {
	base := parseConfiguredAgents()
	return append(base, runtimeAgents...)
}

// removeRuntimeAgent удаляет только runtime-агента (не из YAML/env).
func removeRuntimeAgent(rawURL string) error {
	url := normalizeAgentURL(rawURL)
	if url == "" {
		return errInvalidAgentURL
	}
	loadRuntimeAgents()
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	found := false
	next := runtimeAgents[:0]
	for _, a := range runtimeAgents {
		if normalizeAgentURL(a.URL) == url {
			found = true
			continue
		}
		next = append(next, a)
	}
	if !found {
		return errAgentNotRuntime
	}
	runtimeAgents = next
	return saveRuntimeAgentsLocked()
}

type agentError string

func (e agentError) Error() string { return string(e) }

const (
	errInvalidAgentURL = agentError("invalid agent url")
	errAgentExists     = agentError("agent already registered")
	errAgentNotRuntime = agentError("agent is from config/env; remove it from YAML or NGINX_LENS_AGENTS")
)
