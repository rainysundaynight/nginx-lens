package analyzer

import "github.com/rainysundaynight/nginx-lens/internal/parser"

// ---------- Пустые блоки ----------
// Поиск server/location/upstream без вложенных директив.

// EmptyBlock — пустой блок конфигурации.
type EmptyBlock struct {
	Block string `json:"block"`
	Arg   string `json:"arg,omitempty"`
	File  string `json:"file,omitempty"`
	Line  int    `json:"line,omitempty"`
}

// FindEmptyBlocks находит пустые блоки без вложенных директив.
func FindEmptyBlocks(tree *parser.ConfigTree) []EmptyBlock {
	var empties []EmptyBlock
	for _, item := range Walk(tree) {
		if item.Node.Block != "" && len(item.Node.Directives) == 0 {
			empties = append(empties, EmptyBlock{
				Block: item.Node.Block,
				File:  item.Node.File,
				Line:  item.Node.Line,
				Arg:   item.Node.Arg,
			})
		}
	}
	return empties
}
