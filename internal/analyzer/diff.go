package analyzer

import (
	"fmt"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Сравнение деревьев конфигурации ----------
// Структурный diff двух parsed config trees.

// TreeDiff — отличие между двумя деревьями.
type TreeDiff struct {
	Type   string      `json:"type"`
	Path   []string    `json:"path"`
	Value1 interface{} `json:"value1,omitempty"`
	Value2 interface{} `json:"value2,omitempty"`
}

// DiffTrees сравнивает два дерева директив.
func DiffTrees(tree1, tree2 *parser.ConfigTree) []TreeDiff {
	var diffs []TreeDiff
	diffBlocks(tree1.Directives, tree2.Directives, nil, &diffs)
	return diffs
}

func nodeKey(d parser.Node) string {
	if d.Block != "" {
		return fmt.Sprintf("block:%s:%s", d.Block, d.Arg)
	}
	if d.Directive != "" {
		return fmt.Sprintf("directive:%s:%s", d.Directive, d.Args)
	}
	if d.Upstream != "" {
		return fmt.Sprintf("upstream:%s", d.Upstream)
	}
	return fmt.Sprintf("other:%v", d)
}

func diffBlocks(d1, d2 []parser.Node, path []string, diffs *[]TreeDiff) {
	map1 := make(map[string]parser.Node)
	map2 := make(map[string]parser.Node)
	for _, n := range d1 {
		map1[nodeKey(n)] = n
	}
	for _, n := range d2 {
		map2[nodeKey(n)] = n
	}
	allKeys := make(map[string]struct{})
	for k := range map1 {
		allKeys[k] = struct{}{}
	}
	for k := range map2 {
		allKeys[k] = struct{}{}
	}
	for k := range allKeys {
		v1, ok1 := map1[k]
		v2, ok2 := map2[k]
		p := append(path, k)
		if ok1 && !ok2 {
			*diffs = append(*diffs, TreeDiff{Type: "removed", Path: p, Value1: v1})
		} else if ok2 && !ok1 {
			*diffs = append(*diffs, TreeDiff{Type: "added", Path: p, Value2: v2})
		} else if len(v1.Directives) > 0 && len(v2.Directives) > 0 {
			diffBlocks(v1.Directives, v2.Directives, p, diffs)
		} else if nodeKey(v1) != nodeKey(v2) {
			*diffs = append(*diffs, TreeDiff{Type: "changed", Path: p, Value1: v1, Value2: v2})
		}
	}
}
