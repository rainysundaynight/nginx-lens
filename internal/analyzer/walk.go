package analyzer

import "github.com/rainysundaynight/nginx-lens/internal/parser"

// ---------- Обход дерева конфигурации ----------
// Рекурсивный итератор по узлам parsed nginx config.

// WalkItem — пара (узел, родитель) при обходе дерева.
type WalkItem struct {
	Node   parser.Node
	Parent *parser.Node
}

// Walk рекурсивно обходит дерево директив.
func Walk(tree *parser.ConfigTree) []WalkItem {
	return walkNodes(tree.Directives, nil)
}

// WalkNodes обходит произвольный список узлов.
func WalkNodes(nodes []parser.Node, parent *parser.Node) []WalkItem {
	return walkNodes(nodes, parent)
}

func walkNodes(nodes []parser.Node, parent *parser.Node) []WalkItem {
	var items []WalkItem
	for i := range nodes {
		node := &nodes[i]
		items = append(items, WalkItem{Node: *node, Parent: parent})
		if len(node.Directives) > 0 {
			items = append(items, walkNodes(node.Directives, node)...)
		}
	}
	return items
}
