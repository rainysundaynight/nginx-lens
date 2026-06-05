package webauth

import (
	"net/http"
	"os"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/config"
)

// ---------- Аутентификация web-стека ----------
// Token auth для agent и hub.

const (
	envAgentToken = "NGINX_LENS_AGENT_TOKEN"
	envHubToken   = "NGINX_LENS_HUB_TOKEN"
	envHubAgent   = "NGINX_LENS_HUB_AGENT_TOKEN"
)

// AgentToken возвращает токен агента из env или конфига.
func AgentToken() string {
	if t := os.Getenv(envAgentToken); t != "" {
		return t
	}
	return config.Get().Config.Web.Agent.Token
}

// HubToken возвращает токен hub из env или конфига.
func HubToken() string {
	if t := os.Getenv(envHubToken); t != "" {
		return t
	}
	return config.Get().Config.Web.Hub.Token
}

// HubAgentToken возвращает токен hub→agent.
func HubAgentToken() string {
	if t := os.Getenv(envHubAgent); t != "" {
		return t
	}
	return config.Get().Config.Web.Hub.AgentToken
}

// VerifyAgentToken middleware для agent API.
func VerifyAgentToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := AgentToken()
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !checkToken(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// VerifyHubToken middleware для hub API.
func VerifyHubToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := HubToken()
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !checkToken(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkToken(r *http.Request, expected string) bool {
	if h := r.Header.Get("X-Nginx-Lens-Token"); h == expected {
		return true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == expected {
		return true
	}
	return false
}

// AgentHeaders возвращает заголовки для hub→agent запросов.
func AgentHeaders() map[string]string {
	token := HubAgentToken()
	if token == "" {
		token = AgentToken()
	}
	if token == "" {
		return nil
	}
	return map[string]string{"X-Nginx-Lens-Token": token}
}
