package analyzer

import (
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Конфликты location и listen/server_name ----------
// Поиск пересечений location и дублирующих server-блоков.

// LocationConflict — пересечение location внутри server.
type LocationConflict struct {
	Server    parser.Node `json:"server"`
	Location1 string      `json:"location1"`
	Location2 string      `json:"location2"`
}

// ListenConflict — конфликт listen/server_name между server-блоками.
type ListenConflict struct {
	Server1    parser.Node `json:"server1"`
	Server2    parser.Node `json:"server2"`
	Listen     []string    `json:"listen"`
	ServerName []string    `json:"server_name"`
}

// FindLocationConflicts находит пересекающиеся location в server-блоках.
func FindLocationConflicts(tree *parser.ConfigTree) []LocationConflict {
	var conflicts []LocationConflict
	for _, item := range Walk(tree) {
		if item.Node.Block != "server" {
			continue
		}
		var locations []string
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			if sub.Node.Block == "location" && sub.Node.Arg != "" {
				locations = append(locations, sub.Node.Arg)
			}
		}
		for i := 0; i < len(locations); i++ {
			for j := i + 1; j < len(locations); j++ {
				if locationsConflict(locations[i], locations[j]) {
					conflicts = append(conflicts, LocationConflict{
						Server:    item.Node,
						Location1: locations[i],
						Location2: locations[j],
					})
				}
			}
		}
	}
	return conflicts
}

func locationsConflict(loc1, loc2 string) bool {
	return strings.HasPrefix(loc1, loc2) || strings.HasPrefix(loc2, loc1)
}

// FindListenServerNameConflicts находит конфликтующие listen/server_name.
func FindListenServerNameConflicts(tree *parser.ConfigTree) []ListenConflict {
	type serverInfo struct {
		block      parser.Node
		listen     map[string]struct{}
		serverName map[string]struct{}
	}
	var servers []serverInfo
	for _, item := range Walk(tree) {
		if item.Node.Block != "server" {
			continue
		}
		info := serverInfo{
			block:      item.Node,
			listen:     make(map[string]struct{}),
			serverName: make(map[string]struct{}),
		}
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			if sub.Node.Directive == "listen" {
				info.listen[strings.TrimSpace(sub.Node.Args)] = struct{}{}
			}
			if sub.Node.Directive == "server_name" {
				for _, n := range strings.Fields(sub.Node.Args) {
					info.serverName[n] = struct{}{}
				}
			}
		}
		servers = append(servers, info)
	}
	var conflicts []ListenConflict
	for i := 0; i < len(servers); i++ {
		for j := i + 1; j < len(servers); j++ {
			var commonListen, commonName []string
			for l := range servers[i].listen {
				if _, ok := servers[j].listen[l]; ok {
					commonListen = append(commonListen, l)
				}
			}
			for n := range servers[i].serverName {
				if _, ok := servers[j].serverName[n]; ok {
					commonName = append(commonName, n)
				}
			}
			if len(commonListen) > 0 && len(commonName) > 0 {
				conflicts = append(conflicts, ListenConflict{
					Server1:    servers[i].block,
					Server2:    servers[j].block,
					Listen:     commonListen,
					ServerName: commonName,
				})
			}
		}
	}
	return conflicts
}
