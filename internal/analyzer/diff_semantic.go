package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

// DiffSemantic сравнивает upstream, backend-серверы и proxy_pass-маршруты.
func DiffSemantic(tree1, tree2 *parser.ConfigTree) []TreeDiff {
	var diffs []TreeDiff
	diffs = append(diffs, diffUpstreamBlocks(tree1, tree2)...)
	diffs = append(diffs, diffRouteRefs(tree1, tree2)...)
	if len(diffs) == 0 {
		diffs = DiffTrees(tree1, tree2)
	}
	return diffs
}

func diffUpstreamBlocks(t1, t2 *parser.ConfigTree) []TreeDiff {
	u1, u2 := t1.GetUpstreams(), t2.GetUpstreams()
	keys := stringKeyUnion(mapKeys(u1), mapKeys(u2))
	var diffs []TreeDiff
	for _, name := range keys {
		s1, ok1 := u1[name]
		s2, ok2 := u2[name]
		switch {
		case ok1 && !ok2:
			diffs = append(diffs, TreeDiff{
				Type: "removed", Path: []string{"upstream", name},
				Value1: strings.Join(s1, ", "),
			})
		case ok2 && !ok1:
			diffs = append(diffs, TreeDiff{
				Type: "added", Path: []string{"upstream", name},
				Value2: strings.Join(s2, ", "),
			})
		case !serversEqual(s1, s2):
			diffs = append(diffs, TreeDiff{
				Type: "changed", Path: []string{"upstream", name},
				Value1: strings.Join(s1, ", "),
				Value2: strings.Join(s2, ", "),
			})
		}
	}
	return diffs
}

func diffRouteRefs(t1, t2 *parser.ConfigTree) []TreeDiff {
	r1 := routeRefMap(t1)
	r2 := routeRefMap(t2)
	keys := stringKeyUnion(mapKeys(r1), mapKeys(r2))
	var diffs []TreeDiff
	for _, key := range keys {
		a, ok1 := r1[key]
		b, ok2 := r2[key]
		switch {
		case ok1 && !ok2:
			diffs = append(diffs, TreeDiff{
				Type: "removed", Path: routePath(a),
				Value1: a.ProxyPass,
			})
		case ok2 && !ok1:
			diffs = append(diffs, TreeDiff{
				Type: "added", Path: routePath(b),
				Value2: b.ProxyPass,
			})
		case a.ProxyPass != b.ProxyPass || a.FromDirective != b.FromDirective:
			diffs = append(diffs, TreeDiff{
				Type: "changed", Path: routePath(a),
				Value1: fmt.Sprintf("%s → %s", a.FromDirective, a.ProxyPass),
				Value2: fmt.Sprintf("%s → %s", b.FromDirective, b.ProxyPass),
			})
		}
	}
	return diffs
}

func routeRefMap(tree *parser.ConfigTree) map[string]BlastRadiusEntry {
	names := upstream.CollectKnownUpstreamNames(tree)
	refs := upstream.FindUpstreamReferences(tree, names)
	m := make(map[string]BlastRadiusEntry, len(refs))
	for _, ref := range refs {
		key := routeKey(ref)
		m[key] = BlastRadiusEntry{
			UpstreamName:  ref.UpstreamName,
			ServerName:    ref.ServerName,
			Listen:        ref.Listen,
			Location:      ref.Location,
			FromDirective: ref.FromDirective,
			ProxyPass:     ref.Value,
			ConfigFile:    ref.ConfigFile,
		}
	}
	return m
}

func routeKey(ref upstream.UpstreamRef) string {
	loc := ref.Location
	if loc == "" {
		loc = "/"
	}
	return strings.Join([]string{ref.ServerName, ref.Listen, loc, ref.FromDirective}, "|")
}

func routePath(e BlastRadiusEntry) []string {
	parts := []string{"server"}
	if e.ServerName != "" {
		parts = append(parts, e.ServerName)
	}
	if e.Listen != "" {
		parts = append(parts, "listen:"+e.Listen)
	}
	if e.Location != "" {
		parts = append(parts, "location:"+e.Location)
	}
	return parts
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func stringKeyUnion(a, b []string) []string {
	seen := make(map[string]struct{})
	for _, k := range a {
		seen[k] = struct{}{}
	}
	for _, k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func serversEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}
