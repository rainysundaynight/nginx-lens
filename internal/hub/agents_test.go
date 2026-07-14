package hub

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAgentURL(t *testing.T) {
	assert.Equal(t, "http://10.0.0.1:8088", normalizeAgentURL("http://10.0.0.1:8088/"))
	assert.Equal(t, "", normalizeAgentURL("ftp://x"))
	assert.Equal(t, "", normalizeAgentURL("not-a-url"))
}

func TestAddAndRemoveRuntimeAgent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NGINX_LENS_HUB_AGENTS_FILE", filepath.Join(dir, "agents.json"))
	t.Setenv("NGINX_LENS_AGENTS", "http://only-from-env:8088")

	runtimeMu.Lock()
	runtimeAgents = nil
	runtimeLoaded = false
	runtimeMu.Unlock()
	t.Cleanup(func() {
		runtimeMu.Lock()
		runtimeAgents = nil
		runtimeLoaded = false
		runtimeMu.Unlock()
	})

	ep, err := addRuntimeAgent("http://127.0.0.1:9090", "LOCAL", "test")
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:9090", ep.URL)
	assert.Equal(t, "runtime", ep.Source)

	_, err = addRuntimeAgent("http://127.0.0.1:9090", "", "")
	assert.ErrorIs(t, err, errAgentExists)

	require.NoError(t, removeRuntimeAgent("http://127.0.0.1:9090"))
	_, err = os.Stat(filepath.Join(dir, "agents.json"))
	assert.NoError(t, err)
}

func TestFormatUptimeSec(t *testing.T) {
	assert.Equal(t, "45s", formatUptimeSec(45))
	assert.Equal(t, "2m", formatUptimeSec(120))
	assert.Equal(t, "1h 5m", formatUptimeSec(3900))
	assert.Equal(t, "2d 3h", formatUptimeSec(2*86400+3*3600))
}

func TestKPIDeltasAndAvailability(t *testing.T) {
	historyMu.Lock()
	historySamples = nil
	historyMu.Unlock()
	t.Cleanup(func() {
		historyMu.Lock()
		historySamples = nil
		historyMu.Unlock()
	})

	old := kpiSample{
		At:           time.Now().Add(-10 * time.Minute),
		AgentsOnline: 1,
		Critical:     5,
		Warnings:     2,
		UpstreamPct:  90,
		AgentOnline:  map[string]bool{"http://a": true},
	}
	recordKPISample(old)
	recordKPISample(kpiSample{
		At:           time.Now().Add(-9 * time.Minute),
		AgentsOnline: 1,
		Critical:     5,
		Warnings:     2,
		UpstreamPct:  90,
		AgentOnline:  map[string]bool{"http://a": false},
	})

	prev := sampleNear1h()
	require.NotNil(t, prev)
	cur := kpiSample{AgentsOnline: 2, Critical: 3, Warnings: 4, UpstreamPct: 95}
	d := buildKPIDeltas(cur, prev)
	require.NotNil(t, d)
	assert.Equal(t, "+1", d.AgentsOnline.Value)
	assert.True(t, d.AgentsOnline.Positive)
	assert.Equal(t, "-2", d.CriticalIssues.Value)
	assert.True(t, d.CriticalIssues.Positive)

	avail := agentAvailabilityPct("http://a")
	assert.Equal(t, "50.0%", avail)
}
