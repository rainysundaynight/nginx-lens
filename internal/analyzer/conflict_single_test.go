package analyzer

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestConflictSingleFile(t *testing.T) {
	tree, err := parser.ParseNginxConfig("../../testdata/nginx/conflict-single.conf")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range Walk(tree) {
		if item.Node.Block == "server" {
			t.Logf("server block file=%s line=%d", item.Node.File, item.Node.Line)
		}
	}
	conflicts := FindListenServerNameConflicts(tree)
	for _, c := range conflicts {
		t.Logf("conflict: s1=%s:%d s2=%s:%d listen=%v names=%v", c.Server1.File, c.Server1.Line, c.Server2.File, c.Server2.Line, c.Listen, c.ServerName)
	}
	assert.Empty(t, conflicts)
	for _, item := range Walk(tree) {
		if item.Node.Block == "server" {
			assert.Equal(t, 5, item.Node.Line)
			break
		}
	}
}

func TestConflictDoubleInclude(t *testing.T) {
	tree, err := parser.ParseNginxConfig("../../testdata/nginx/conflict-double-include.conf")
	if err != nil {
		t.Fatal(err)
	}
	var servers []parser.Node
	for _, item := range Walk(tree) {
		if item.Node.Block == "server" {
			servers = append(servers, item.Node)
			t.Logf("server file=%s line=%d", item.Node.File, item.Node.Line)
		}
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 server nodes from double include, got %d", len(servers))
	}
	conflicts := FindListenServerNameConflicts(tree)
	assert.Empty(t, conflicts, "double include of identical vhost should not report listen/server_name conflict")
}

func TestConflictDefaultPlusNamedVhost(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{
		{
			Block: "server",
			File:  "default.conf", Line: 1,
			Directives: []parser.Node{
				{Directive: "listen", Args: "80 default_server"},
				{Directive: "server_name", Args: "_"},
			},
		},
		{
			Block: "server",
			File:  "001-test.conf", Line: 5,
			Directives: []parser.Node{
				{Directive: "listen", Args: "80"},
				{Directive: "server_name", Args: "test.clinty.app"},
			},
		},
	}, nil)
	conflicts := FindListenServerNameConflicts(tree)
	assert.Empty(t, conflicts, "default _ vhost + named vhost on same port is valid nginx")
}

func TestConflictRealDuplicate(t *testing.T) {
	tree := parser.NewConfigTree([]parser.Node{
		{
			Block: "server",
			File:  "a.conf", Line: 1,
			Directives: []parser.Node{
				{Directive: "listen", Args: "80"},
				{Directive: "server_name", Args: "example.com"},
			},
		},
		{
			Block: "server",
			File:  "b.conf", Line: 10,
			Directives: []parser.Node{
				{Directive: "listen", Args: "80"},
				{Directive: "server_name", Args: "example.com"},
			},
		},
	}, nil)
	conflicts := FindListenServerNameConflicts(tree)
	assert.Len(t, conflicts, 1)
	assert.Contains(t, listenConflictDesc(conflicts[0]), "example.com")
}
