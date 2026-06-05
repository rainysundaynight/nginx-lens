package analyzer

import "github.com/rainysundaynight/nginx-lens/internal/parser"

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
		if item.Node.Block == "server" {
			for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
				if sub.Node.Block == "location" {
					locations = append(locations, locEntry{server: item.Node, location: sub.Node})
				}
			}
		}
	}
	for _, item := range Walk(tree) {
		for _, key := range []string{"proxy_pass", "rewrite", "try_files"} {
			if item.Node.Directive != key {
				continue
			}
			for _, l := range locations {
				loc := l.location.Arg
				if loc != "" && containsStr(item.Node.Args, loc) {
					k := l.server.Arg + "\x00" + loc
					used[k] = struct{}{}
				}
			}
		}
	}
	var dead []DeadLocation
	for _, l := range locations {
		k := l.server.Arg + "\x00" + l.location.Arg
		if _, ok := used[k]; !ok {
			dead = append(dead, DeadLocation{Server: l.server, Location: l.location})
		}
	}
	return dead
}

func containsStr(s, substr string) bool {
	return len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
