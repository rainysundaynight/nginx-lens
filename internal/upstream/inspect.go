package upstream

import (
	"regexp"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Инспекция upstream ----------
// Ссылки на upstream и разбор server-строк.

// UpstreamRef — ссылка на named upstream.
type UpstreamRef struct {
	UpstreamName   string `json:"upstream_name"`
	FromDirective  string `json:"from_directive"`
	Value          string `json:"value"`
	ServerName     string `json:"server_name"`
	Listen         string `json:"listen"`
	Location       string `json:"location"`
	ConfigFile     string `json:"config_file"`
	IsStream       bool   `json:"is_stream"`
}

// UpstreamBlock — сводка по upstream-блоку.
type UpstreamBlock struct {
	Name      string                  `json:"name"`
	Servers   []string                `json:"servers"`
	Options   []parser.DirectiveOption `json:"options"`
	File      string                  `json:"file"`
	Refs      []UpstreamRef           `json:"refs,omitempty"`
}

var passDirectives = []string{"proxy_pass", "fastcgi_pass", "uwsgi_pass", "scgi_pass", "grpc_pass", "memcached_pass"}

// FindUpstreamBlocks возвращает все upstream-блоки из дерева.
func FindUpstreamBlocks(tree *parser.ConfigTree) []UpstreamBlock {
	var blocks []UpstreamBlock
	for _, item := range walkAll(tree.Directives) {
		if item.Upstream != "" {
			blocks = append(blocks, UpstreamBlock{
				Name:    item.Upstream,
				Servers: item.Servers,
				Options: item.Options,
				File:    item.File,
			})
		}
	}
	return blocks
}

// FindUpstreamReferences находит ссылки на upstream в конфигурации.
func FindUpstreamReferences(tree *parser.ConfigTree, knownNames map[string]struct{}) []UpstreamRef {
	var refs []UpstreamRef
	for _, item := range walkAll(tree.Directives) {
		if item.Block == "server" || item.Block == "location" {
			serverName, listen, location := "", "", ""
			if item.Block == "location" {
				location = item.Arg
			}
			for _, sub := range item.Directives {
				if sub.Directive == "server_name" {
					serverName = sub.Args
				}
				if sub.Directive == "listen" {
					listen = sub.Args
				}
			}
			for _, sub := range item.Directives {
				for _, passDir := range passDirectives {
					if sub.Directive != passDir {
						continue
					}
					name := extractUpstreamName(sub.Args)
					if name != "" {
						if _, known := knownNames[name]; known {
							refs = append(refs, UpstreamRef{
								UpstreamName:  name,
								FromDirective: passDir,
								Value:         sub.Args,
								ServerName:    serverName,
								Listen:        listen,
								Location:      location,
								ConfigFile:    sub.File,
							})
						}
					}
				}
			}
		}
	}
	return DedupeRefs(refs)
}

// CollectKnownUpstreamNames собирает имена всех upstream-блоков.
func CollectKnownUpstreamNames(tree *parser.ConfigTree) map[string]struct{} {
	names := make(map[string]struct{})
	for _, item := range walkAll(tree.Directives) {
		if item.Upstream != "" {
			names[item.Upstream] = struct{}{}
		}
	}
	return names
}

// ParseServerOptions разбирает параметры server-строки upstream.
func ParseServerOptions(serverLine string) map[string]string {
	opts := make(map[string]string)
	parts := strings.Fields(serverLine)
	if len(parts) == 0 {
		return opts
	}
	for _, p := range parts[1:] {
		if strings.Contains(p, "=") {
			kv := strings.SplitN(p, "=", 2)
			opts[kv[0]] = kv[1]
		} else {
			opts[p] = "true"
		}
	}
	return opts
}

func extractUpstreamName(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		rest := strings.SplitN(value, "://", 2)[1]
		if idx := strings.Index(rest, "/"); idx >= 0 {
			rest = rest[:idx]
		}
		return rest
	}
	if strings.HasPrefix(value, "grpc://") || strings.HasPrefix(value, "grpcs://") {
		for _, pfx := range []string{"grpc://", "grpcs://"} {
			if strings.HasPrefix(value, pfx) {
				rest := strings.TrimPrefix(value, pfx)
				if idx := strings.Index(rest, "/"); idx >= 0 {
					rest = rest[:idx]
				}
				return rest
			}
		}
	}
	re := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)
	if re.MatchString(value) {
		return value
	}
	return ""
}

func walkAll(nodes []parser.Node) []parser.Node {
	var result []parser.Node
	for _, n := range nodes {
		result = append(result, n)
		if len(n.Directives) > 0 {
			result = append(result, walkAll(n.Directives)...)
		}
	}
	return result
}
