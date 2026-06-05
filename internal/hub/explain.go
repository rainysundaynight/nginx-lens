package hub

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/webauth"
)

// handleExplain проксирует /explain с агента по URL маршрута.
func handleExplain(w http.ResponseWriter, r *http.Request) {
	agent := strings.TrimSpace(r.URL.Query().Get("agent"))
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if agent == "" || rawURL == "" {
		http.Error(w, "agent and url required", http.StatusBadRequest)
		return
	}
	target := strings.TrimRight(agent, "/") + "/explain?url=" + url.QueryEscape(rawURL)
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for k, v := range webauth.AgentHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
