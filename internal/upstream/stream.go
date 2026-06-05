package upstream

import (
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Stream upstream ----------
// Blast-radius для stream { } без цикла import analyzer.

// StreamEntry — stream upstream → listen.
type StreamEntry struct {
	UpstreamName  string `json:"upstream_name"`
	Listen        string `json:"listen"`
	FromDirective string `json:"from_directive"`
	ProxyPass     string `json:"proxy_pass"`
	ConfigFile    string `json:"config_file"`
}

// StreamGraph — граф stream-зависимостей.
type StreamGraph map[string][]StreamEntry

// BuildStreamGraph строит dependency graph для stream-контекста.
func BuildStreamGraph(tree *parser.ConfigTree) StreamGraph {
	names := CollectKnownUpstreamNames(tree)
	refs := findStreamReferences(tree.Directives, false, names)
	graph := make(StreamGraph)
	for _, ref := range refs {
		graph[ref.UpstreamName] = append(graph[ref.UpstreamName], StreamEntry{
			UpstreamName:  ref.UpstreamName,
			Listen:        ref.Listen,
			FromDirective: ref.FromDirective,
			ProxyPass:     ref.Value,
			ConfigFile:    ref.ConfigFile,
		})
	}
	for k, v := range graph {
		graph[k] = dedupeStreamEntries(v)
	}
	return graph
}

// StreamReferencedNames возвращает upstream, используемые в stream { }.
func StreamReferencedNames(tree *parser.ConfigTree) map[string]struct{} {
	graph := BuildStreamGraph(tree)
	names := make(map[string]struct{}, len(graph))
	for name := range graph {
		names[name] = struct{}{}
	}
	return names
}

func findStreamReferences(nodes []parser.Node, inStream bool, names map[string]struct{}) []UpstreamRef {
	var refs []UpstreamRef
	for _, n := range nodes {
		streamCtx := inStream || n.Block == "stream"
		if n.Block == "server" && streamCtx {
			listen := ""
			for _, sub := range n.Directives {
				if sub.Directive == "listen" {
					listen = sub.Args
				}
			}
			for _, sub := range n.Directives {
				for _, passDir := range passDirectives {
					if sub.Directive != passDir {
						continue
					}
					name := extractUpstreamName(sub.Args)
					if name != "" {
						if _, known := names[name]; known {
							refs = append(refs, UpstreamRef{
								UpstreamName:  name,
								FromDirective: passDir,
								Value:         sub.Args,
								Listen:        listen,
								ConfigFile:    sub.File,
								IsStream:      true,
							})
						}
					}
				}
			}
		}
		if len(n.Directives) > 0 {
			refs = append(refs, findStreamReferences(n.Directives, streamCtx, names)...)
		}
	}
	return DedupeRefs(refs)
}

func dedupeStreamEntries(entries []StreamEntry) []StreamEntry {
	seen := make(map[string]struct{})
	var out []StreamEntry
	for _, e := range entries {
		key := e.UpstreamName + "\x00" + e.Listen + "\x00" + e.ProxyPass
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, e)
	}
	return out
}
