package analyzer

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Мёртвые location ----------
// Location без ссылок в proxy_pass/rewrite/try_files.

// DeadLocation — неиспользуемый location.
type DeadLocation struct {
	Server   parser.Node `json:"server"`
	Location parser.Node `json:"location"`
}

// FindDeadLocations находит location без ссылок в proxy_pass/rewrite/try_files.
func FindDeadLocations(tree *parser.ConfigTree) []DeadLocation {
	type locEntry struct {
		server   parser.Node
		location parser.Node
	}
	var locations []locEntry
	used := make(map[string]struct{})

	for _, item := range Walk(tree) {
		if item.Node.Block != "server" {
			continue
		}
		scope := ServerScopeKey(item.Node)
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			if sub.Node.Block == "location" {
				locations = append(locations, locEntry{server: item.Node, location: sub.Node})
			}
		}
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			for _, key := range []string{"proxy_pass", "rewrite", "try_files"} {
				if sub.Node.Directive != key {
					continue
				}
				for _, l := range locations {
					if ServerScopeKey(l.server) != scope {
						continue
					}
					loc := l.location.Arg
					if loc == "" || loc == "/" {
						continue
					}
					if locationReferenced(sub.Node.Args, loc) {
						used[scope+"\x00"+loc] = struct{}{}
					}
				}
			}
		}
	}
	var dead []DeadLocation
	for _, l := range locations {
		k := ServerScopeKey(l.server) + "\x00" + l.location.Arg
		if _, ok := used[k]; !ok {
			dead = append(dead, DeadLocation{Server: l.server, Location: l.location})
		}
	}
	return dead
}

func locationReferenced(args, loc string) bool {
	if args == loc {
		return true
	}
	for _, tok := range strings.Fields(args) {
		if strings.Contains(tok, "/") && (tok == loc || strings.HasPrefix(tok, loc+"/")) {
			return true
		}
	}
	return false
}
