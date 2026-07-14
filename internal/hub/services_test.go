package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildServicesAppsFromAccess(t *testing.T) {
	results := []map[string]interface{}{
		{
			"agent":  "http://10.0.0.1:8088",
			"online": true,
			"label":  "edge-1",
			"snapshot": map[string]interface{}{
				"access_stats": map[string]interface{}{
					"total_requests": float64(1000),
					"status_5xx":     float64(50),
					"by_upstream": map[string]interface{}{
						"10.0.0.5:8080": map[string]interface{}{
							"requests":   float64(700),
							"status_5xx": float64(40),
						},
						"_direct": map[string]interface{}{
							"requests":   float64(300),
							"status_5xx": float64(10),
						},
					},
					"top_paths": []interface{}{
						map[string]interface{}{"path": "/api/v1", "requests": float64(400), "status_5xx": float64(20)},
						map[string]interface{}{"path": "/static", "requests": float64(200), "status_5xx": float64(0)},
					},
				},
			},
		},
	}
	svc := buildServicesApps(results)
	require.True(t, svc.HasData)
	assert.Equal(t, 1000, svc.TotalRequests)
	assert.Equal(t, 1, svc.UniqueServices)
	require.NotEmpty(t, svc.Upstreams)
	assert.Equal(t, "10.0.0.5:8080", svc.Upstreams[0].Name)
	assert.Equal(t, 700, svc.Upstreams[0].Count)
	require.Len(t, svc.Endpoints, 2)
	assert.Equal(t, "/api/v1", svc.Endpoints[0].Name)
	assert.InDelta(t, 70.0, svc.UpstreamSharePct, 0.1)
	assert.InDelta(t, 5.0, svc.ErrorSharePct, 0.1)
	require.NotEmpty(t, svc.Quality)
	assert.Equal(t, 660, svc.Quality[0].Ok)
	assert.Equal(t, 40, svc.Quality[0].Errors)
}
