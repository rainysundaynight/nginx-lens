package analyzer

import "github.com/rainysundaynight/nginx-lens/internal/parser"

// ---------- Дублирующиеся директивы ----------
// Поиск повторяющихся директив на одном уровне блока.

// Duplicate — дублирующаяся директива.
type Duplicate struct {
	Block     parser.Node `json:"block"`
	Directive string      `json:"directive"`
	Args      string      `json:"args"`
	Count     int         `json:"count"`
	Location  string      `json:"location,omitempty"`
}

// FindDuplicateDirectives находит дубли директив внутри блоков.
func FindDuplicateDirectives(tree *parser.ConfigTree) []Duplicate {
	var duplicates []Duplicate
	for _, item := range Walk(tree) {
		if len(item.Node.Directives) == 0 {
			continue
		}
		seen := make(map[string]int)
		for _, sub := range item.Node.Directives {
			if sub.Directive == "" {
				continue
			}
			key := sub.Directive + "\x00" + sub.Args
			seen[key]++
		}
		for key, count := range seen {
			if count > 1 {
				parts := splitKey(key)
				duplicates = append(duplicates, Duplicate{
					Block:     item.Node,
					Directive: parts[0],
					Args:      parts[1],
					Count:     count,
					Location:  item.Node.Arg,
				})
			}
		}
	}
	return duplicates
}

func splitKey(key string) [2]string {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			return [2]string{key[:i], key[i+1:]}
		}
	}
	return [2]string{key, ""}
}
