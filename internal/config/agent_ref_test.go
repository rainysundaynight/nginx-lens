package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAgentRefUnmarshalScalarAndObject(t *testing.T) {
	var cfg struct {
		Agents []AgentRef `yaml:"agents"`
	}
	raw := []byte(`
agents:
  - http://localhost:8088
  - url: http://10.0.0.1:8088
    region: EU-WEST
    name: lon-prod
`)
	require.NoError(t, yaml.Unmarshal(raw, &cfg))
	require.Len(t, cfg.Agents, 2)
	assert.Equal(t, "http://localhost:8088", cfg.Agents[0].URL)
	assert.Equal(t, "http://10.0.0.1:8088", cfg.Agents[1].URL)
	assert.Equal(t, "EU-WEST", cfg.Agents[1].Region)
	assert.Equal(t, "lon-prod", cfg.Agents[1].Name)
}
