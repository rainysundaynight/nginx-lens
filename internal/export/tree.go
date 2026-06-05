package export

import (
	"fmt"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Визуализация дерева ----------
// Markdown, HTML и текстовое дерево конфигурации.

// RenderTreeMarkdown рендерит дерево в Markdown.
func RenderTreeMarkdown(nodes []parser.Node, indent int) string {
	var b strings.Builder
	prefix := strings.Repeat("  ", indent)
	for _, n := range nodes {
		switch {
		case n.Upstream != "":
			b.WriteString(fmt.Sprintf("%s- upstream %s\n", prefix, n.Upstream))
			for _, s := range n.Servers {
				b.WriteString(fmt.Sprintf("%s  - server %s\n", prefix, s))
			}
		case n.Block != "":
			b.WriteString(fmt.Sprintf("%s- %s %s\n", prefix, n.Block, n.Arg))
			b.WriteString(RenderTreeMarkdown(n.Directives, indent+1))
		case n.Directive != "":
			b.WriteString(fmt.Sprintf("%s- %s %s\n", prefix, n.Directive, n.Args))
		}
	}
	return b.String()
}

// RenderTreeHTML рендерит дерево в HTML.
func RenderTreeHTML(nodes []parser.Node) string {
	var b strings.Builder
	b.WriteString("<ul>")
	renderTreeHTMLNodes(&b, nodes)
	b.WriteString("</ul>")
	return b.String()
}

func renderTreeHTMLNodes(b *strings.Builder, nodes []parser.Node) {
	for _, n := range nodes {
		b.WriteString("<li>")
		switch {
		case n.Upstream != "":
			b.WriteString(fmt.Sprintf("<b>upstream %s</b>", n.Upstream))
			if len(n.Servers) > 0 {
				b.WriteString("<ul>")
				for _, s := range n.Servers {
					b.WriteString(fmt.Sprintf("<li>server %s</li>", s))
				}
				b.WriteString("</ul>")
			}
		case n.Block != "":
			b.WriteString(fmt.Sprintf("<b>%s</b> %s", n.Block, n.Arg))
			if len(n.Directives) > 0 {
				b.WriteString("<ul>")
				renderTreeHTMLNodes(b, n.Directives)
				b.WriteString("</ul>")
			}
		case n.Directive != "":
			b.WriteString(fmt.Sprintf("%s %s", n.Directive, n.Args))
		}
		b.WriteString("</li>")
	}
}

// RenderTreeText рендерит дерево в текстовом виде.
func RenderTreeText(nodes []parser.Node, prefix string, isLast bool) string {
	var b strings.Builder
	for i, n := range nodes {
		last := i == len(nodes)-1
		connector := "├── "
		if last {
			connector = "└── "
		}
		switch {
		case n.Upstream != "":
			b.WriteString(fmt.Sprintf("%s%supstream %s\n", prefix, connector, n.Upstream))
		case n.Block != "":
			b.WriteString(fmt.Sprintf("%s%s%s %s\n", prefix, connector, n.Block, n.Arg))
			childPrefix := prefix
			if last {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
			b.WriteString(RenderTreeText(n.Directives, childPrefix, last))
		case n.Directive != "":
			b.WriteString(fmt.Sprintf("%s%s%s %s\n", prefix, connector, n.Directive, n.Args))
		}
	}
	return b.String()
}
