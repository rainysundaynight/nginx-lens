package hub

import (
	"net/http"
	"net/http/httptest"
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
	assert.Contains(t, w.Body.String(), "OBSERVABILITY HUB")
	assert.Contains(t, w.Body.String(), "sidebar-nav")
	assert.Contains(t, w.Body.String(), "BUILT FOR INFRASTRUCTURE PRECISION")
}

func TestHubState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hub/state", nil)
	w := httptest.NewRecorder()
	NewRouter().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"snapshots"`)
	assert.Contains(t, w.Body.String(), `"kpi"`)
}
