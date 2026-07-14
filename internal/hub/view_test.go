package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildHubStateOffline(t *testing.T) {
	results := []map[string]interface{}{
		{"agent": "http://127.0.0.1:8090", "online": false, "error": "connection refused"},
	}
	state := BuildHubState(results, "2.3.0", 30)
	assert.Equal(t, 0, state.Meta.AgentsOnline)
	assert.Equal(t, 1, state.Meta.AgentsTotal)
	assert.Equal(t, "OFFLINE", state.Meta.SystemStatus)
	assert.Len(t, state.Snapshots, 1)
	assert.Equal(t, "offline", state.Snapshots[0].Status)
}

func TestBuildHubStateOnline(t *testing.T) {
	results := []map[string]interface{}{
		{
			"agent":  "http://10.0.0.1:8090",
			"online": true,
			"snapshot": map[string]interface{}{
				"score": map[string]interface{}{
					"total": float64(85),
					"categories": []interface{}{
						map[string]interface{}{"name": "security", "score": float64(90), "issues": float64(2)},
						map[string]interface{}{"name": "performance", "score": float64(84), "issues": float64(7)},
					},
				},
				"analyze": map[string]interface{}{
					"summary": map[string]interface{}{"high": float64(1), "medium": float64(2), "low": float64(3)},
					"issues":  []interface{}{},
				},
				"nginx_build": map[string]interface{}{"version": "1.25.3"},
				"upstreams_health": map[string]interface{}{
					"backend": []interface{}{
						map[string]interface{}{"address": "127.0.0.1:8080", "healthy": true},
					},
				},
			},
		},
	}
	state := BuildHubState(results, "2.3.0", 30)
	assert.Equal(t, 1, state.Meta.AgentsOnline)
	assert.Equal(t, "10.0.0.1-8090", state.Snapshots[0].ID)
	assert.Equal(t, 85, state.Snapshots[0].ConfigScore)
	assert.Equal(t, 2, state.Snapshots[0].IssuesBreakdown["security"])
	assert.Equal(t, 7, state.Snapshots[0].IssuesBreakdown["performance"])
	assert.Equal(t, 1, state.Severity.High)
	assert.Equal(t, "warning", state.Snapshots[0].Status)
}

func TestAgentSlug(t *testing.T) {
	assert.Equal(t, "127.0.0.1-8090", agentSlug("http://127.0.0.1:8090"))
}

func TestSplitConfigureArgs(t *testing.T) {
	rows := splitConfigureArgs("--prefix=/etc/nginx --with-http_ssl_module --build=x5_foodtech_gitlab_ID_4492060")
	assert.Equal(t, []HubKV{
		{Name: "prefix", Value: "/etc/nginx"},
		{Name: "with-http_ssl_module", Value: "yes"},
		{Name: "build", Value: "x5_foodtech_gitlab_ID_4492060"},
	}, rows)
}

func TestParseBuildSplitsConfigure(t *testing.T) {
	rows := parseBuild(map[string]interface{}{
		"built_by":        "gcc",
		"configure_args":  "--build=x5_foodtech_gitlab_ID_4492060 --with-pcre-jit",
		"openssl":         "OpenSSL 3.0.0",
	})
	assert.Equal(t, []HubKV{
		{Name: "Built by", Value: "gcc"},
		{Name: "OpenSSL", Value: "OpenSSL 3.0.0"},
		{Name: "build", Value: "x5_foodtech_gitlab_ID_4492060"},
		{Name: "with-pcre-jit", Value: "yes"},
	}, rows)
}
