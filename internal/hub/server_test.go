package hub

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHubHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHubAgents(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHubDashboard(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
  assert.Contains(t, w.Body.String(), "nginx-lens")
	assert.Contains(t, w.Body.String(), "sidebar-nav")
	assert.Contains(t, w.Body.String(), "header-search")
	assert.Contains(t, w.Body.String(), "btn-refresh")
	assert.Contains(t, w.Body.String(), "auth-gate")
	assert.Contains(t, w.Body.String(), "NGINX_LENS_AUTH_REQUIRED")
}

func TestHubAuthStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth", nil)
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"required"`)
}

func TestHubState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hub/state", nil)
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"snapshots"`)
	assert.Contains(t, w.Body.String(), `"kpi"`)
}

func TestAgentHTTPErrorIncludesBody(t *testing.T) {
	resp := &http.Response{
		Status:     "500 Internal Server Error",
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("nginx.conf не найден: /etc/nginx/nginx.conf\n")),
	}
	got := agentHTTPError(resp)
	assert.Contains(t, got, "500 Internal Server Error")
	assert.Contains(t, got, "nginx.conf не найден")
}

func TestAgentHTTPErrorEmptyBody(t *testing.T) {
	resp := &http.Response{
		Status:     "502 Bad Gateway",
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(strings.NewReader("   ")),
	}
	assert.Equal(t, "502 Bad Gateway", agentHTTPError(resp))
}
