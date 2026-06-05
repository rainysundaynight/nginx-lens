package cli

import (
	"fmt"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Единый табличный вывод CLI ----------
// Секции, таблицы и статусы для всех команд в режиме output.format: table.

const tableRuleWidth = 72

func stringsRepeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

func visibleLen(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] == '\033' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		n++
		i++
	}
	return n
}

func padVisible(s string, width int) string {
	if width <= 0 {
		return s
	}
	if pad := width - visibleLen(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func printRule(st styler) {
	fmt.Println(st.gray(stringsRepeat("─", tableRuleWidth)))
}

func printSection(st styler, title string) {
	fmt.Println()
	fmt.Println(st.header(title))
	printRule(st)
}

func printGroup(st styler, title string) {
	fmt.Printf("\n%s\n", st.header(title))
}

func printTable(st styler, widths []int, headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	hdr := make([]string, len(headers))
	for i, h := range headers {
		hdr[i] = st.bold(h)
	}
	fmt.Println(formatTableRow(widths, hdr))
	printRule(st)
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		fmt.Println(formatTableRow(widths, cells))
	}
}

func formatTableRow(widths []int, cells []string) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		if i < len(widths) && widths[i] > 0 {
			parts[i] = padVisible(c, widths[i])
		} else {
			parts[i] = c
		}
	}
	return strings.Join(parts, " ")
}

func printKVTable(st styler, rows [][2]string) {
	maxKey := 8
	for _, r := range rows {
		if l := len(r[0]); l > maxKey {
			maxKey = l
		}
	}
	for _, r := range rows {
		if r[0] == "" && r[1] == "" {
			continue
		}
		fmt.Printf("  %s  %s\n", st.cyan(padVisible(r[0], maxKey)), r[1])
	}
}

func printSummary(st styler, format string, args ...interface{}) {
	fmt.Println(st.gray(fmt.Sprintf(format, args...)))
}

func printIssue(st styler, severity, typ, message, location string) {
	fmt.Printf("[%s] %s: %s\n", st.severity(severity), st.cyan(typ), message)
	if location != "" {
		fmt.Printf("  %s %s\n", st.gray("at"), st.gray(location))
	}
}

func statusLabel(st styler, ok bool) string {
	if ok {
		return st.green("OK")
	}
	return st.red("FAIL")
}

func printEmptyOK(st styler, msg string) {
	fmt.Println(st.ok(msg))
}

func nodeDirective(node *parser.Node, directive string) string {
	if node == nil {
		return ""
	}
	for _, d := range node.Directives {
		if d.Directive == directive {
			return d.Args
		}
	}
	return ""
}
