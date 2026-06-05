package analyzer

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRoute(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Directive: "listen", Args: "80"},
			{Directive: "server_name", Args: "example.com"},
			{Block: "location", Arg: "/api", Directives: []parser.Node{
				{Directive: "proxy_pass", Args: "http://backend"},
			}},
		},
	}}, nil)
	result := FindRoute(tree, "http://example.com/api/users")
	require.NotNil(t, result)
	assert.NotNil(t, result.Server)
	assert.NotNil(t, result.Location)
	assert.Equal(t, "http://backend", result.ProxyPass)
}
