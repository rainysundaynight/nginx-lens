package analyzer

import "github.com/rainysundaynight/nginx-lens/internal/parser"

// ---------- Обход дерева конфигурации ----------
// Рекурсивный итератор по узлам parsed nginx config.

// WalkItem — пара (узел, родитель) при обходе дерева.
type WalkItem struct {
	Node      parser.Node
	Parent    *parser.Node
	Ancestors []*parser.Node
}

// Walk рекурсивно обходит дерево директив.
func Walk(tree *parser.ConfigTree) []WalkItem {
	return walkNodes(tree.Directives, nil, nil)
}

// WalkNodes обходит произвольный список узлов.
func WalkNodes(nodes []parser.Node, parent *parser.Node) []WalkItem {
	var ancestors []*parser.Node
	if parent != nil {
		ancestors = []*parser.Node{parent}
	}
	return walkNodes(nodes, parent, ancestors)
}

func walkNodes(nodes []parser.Node, parent *parser.Node, ancestors []*parser.Node) []WalkItem {
	var items []WalkItem
	for i := range nodes {
		node := &nodes[i]
		items = append(items, WalkItem{Node: *node, Parent: parent, Ancestors: ancestors})
		childAncestors := ancestors
		if parent != nil {
			childAncestors = append(append([]*parser.Node{}, ancestors...), parent)
		}
		if len(node.Directives) > 0 {
			items = append(items, walkNodes(node.Directives, node, childAncestors)...)
		}
	}
	return items
}
