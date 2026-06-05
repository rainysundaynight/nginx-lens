package webauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyAgentTokenEmptyAllows(t *testing.T) {
	t.Setenv("NGINX_LENS_AGENT_TOKEN", "")
	called := false
	h := VerifyAgentToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("expected pass-through, code=%d called=%v", rec.Code, called)
	}
}

func TestVerifyAgentTokenRejects(t *testing.T) {
	t.Setenv("NGINX_LENS_AGENT_TOKEN", "secret")
	h := VerifyAgentToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestVerifyAgentTokenAcceptsHeader(t *testing.T) {
	t.Setenv("NGINX_LENS_AGENT_TOKEN", "secret")
	h := VerifyAgentToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Nginx-Lens-Token", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestVerifyHubTokenBearer(t *testing.T) {
	t.Setenv("NGINX_LENS_HUB_TOKEN", "hub-secret")
	h := VerifyHubToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer hub-secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestAgentHeadersFromEnv(t *testing.T) {
	t.Setenv("NGINX_LENS_HUB_AGENT_TOKEN", "agent-token")
	h := AgentHeaders()
	if h["X-Nginx-Lens-Token"] != "agent-token" {
		t.Fatalf("headers=%v", h)
	}
}

func TestAgentTokenFromEnv(t *testing.T) {
	t.Setenv("NGINX_LENS_AGENT_TOKEN", "from-env")
	if got := AgentToken(); got != "from-env" {
		t.Fatalf("got %q", got)
	}
}
