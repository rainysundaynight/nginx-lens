package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ---------- Дерево include ----------
// Построение дерева include, поиск циклов и shadowing.

// IncludeTree — дерево include-файлов.
type IncludeTree map[string]interface{}

// BuildIncludeTree строит дерево include начиная с path.
func BuildIncludeTree(path string, visited map[string]bool) IncludeTree {
	if visited == nil {
		visited = make(map[string]bool)
	}
	absPath, _ := filepath.Abs(path)
	if visited[absPath] {
		return IncludeTree{absPath: "cycle"}
	}
	visited[absPath] = true

	f, err := os.Open(absPath)
	if err != nil {
		return IncludeTree{absPath: "not_found"}
	}
	defer f.Close()

	var includes []IncludeTree
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "include ") {
			continue
		}
		pattern := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "include "), ";"))
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(filepath.Dir(absPath), pattern)
		}
		matches, _ := filepath.Glob(pattern)
		for _, incPath := range matches {
			newVisited := make(map[string]bool)
			for k, v := range visited {
				newVisited[k] = v
			}
			includes = append(includes, BuildIncludeTree(incPath, newVisited))
		}
	}
	return IncludeTree{absPath: includes}
}

// FindIncludeCycles находит циклы include в дереве.
func FindIncludeCycles(tree IncludeTree, stack []string) [][]string {
	var cycles [][]string
	for k, v := range tree {
		if v == "cycle" {
			cycles = append(cycles, append(stack, k))
		} else if subs, ok := v.([]IncludeTree); ok {
			for _, sub := range subs {
				cycles = append(cycles, FindIncludeCycles(sub, append(stack, k))...)
			}
		}
	}
	return cycles
}

// IncludeShadow — переопределение директивы в include.
type IncludeShadow struct {
	File      string `json:"file"`
	Directive string `json:"directive"`
	Value     string `json:"value"`
}

// FindIncludeShadowing находит переопределения директивы в include-файлах.
func FindIncludeShadowing(tree IncludeTree, directive string) []IncludeShadow {
	var found []IncludeShadow
	var walk func(IncludeTree)
	walk = func(t IncludeTree) {
		for k, v := range t {
			if subs, ok := v.([]IncludeTree); ok {
				f, err := os.Open(k)
				if err == nil {
					scanner := bufio.NewScanner(f)
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if strings.HasPrefix(line, directive+" ") {
							found = append(found, IncludeShadow{File: k, Directive: directive, Value: line})
						}
					}
					f.Close()
				}
				for _, sub := range subs {
					walk(sub)
				}
			}
		}
	}
	walk(tree)
	return found
}
