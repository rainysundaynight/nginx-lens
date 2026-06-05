package analyzer

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestFindLocationConflicts(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Block: "location", Arg: "/api"},
			{Block: "location", Arg: "/api/v1"},
		},
	}}, nil)
	conflicts := FindLocationConflicts(tree)
	assert.NotEmpty(t, conflicts)
}

func TestFindListenServerNameConflicts(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Directive: "listen", Args: "80"},
			{Directive: "server_name", Args: "example.com"},
		},
	}, {
		Block: "server",
		Directives: []parser.Node{
			{Directive: "listen", Args: "80"},
			{Directive: "server_name", Args: "example.com"},
		},
	}}, nil)
	conflicts := FindListenServerNameConflicts(tree)
	assert.NotEmpty(t, conflicts)
}
