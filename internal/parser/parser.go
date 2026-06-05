package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ---------- Регулярные выражения парсера ----------
// Паттерны для разбора директив и блоков nginx.

var (
	reUpstream     = regexp.MustCompile(`^upstream\s+(\S+)\s*\{`)
	reBlock        = regexp.MustCompile(`^(\S+)\s*(\S+)?\s*\{`)
	reDirective    = regexp.MustCompile(`^(\S+)\s+([^;]+);`)
	reServerLine   = regexp.MustCompile(`^server\s+([^;]+);`)
	reUpstreamDir  = regexp.MustCompile(`^(\S+)(?:\s+(.+?))?\s*;`)
)

// ParseNginxConfig парсит nginx.conf и возвращает дерево конфигурации.
func ParseNginxConfig(path string) (*ConfigTree, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(absPath)

	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	directives, upstreams := parseBlock(lines, baseDir, absPath)
	return NewConfigTree(directives, upstreams), nil
}

// stripComments удаляет комментарии из строки конфигурации.
func stripComments(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

// parseUpstreamBlockLines разбирает тело upstream { ... }.
func parseUpstreamBlockLines(blockLines []string) ([]string, []DirectiveOption) {
	var servers []string
	var options []DirectiveOption
	for _, bl := range blockLines {
		bl = strings.TrimSpace(bl)
		if bl == "" {
			continue
		}
		if m := reServerLine.FindStringSubmatch(bl); m != nil {
			servers = append(servers, strings.TrimSpace(m[1]))
			continue
		}
		if m := reUpstreamDir.FindStringSubmatch(bl); m != nil {
			args := ""
			if len(m) > 2 && m[2] != "" {
				args = strings.TrimSpace(m[2])
			}
			options = append(options, DirectiveOption{Name: m[1], Args: args})
		}
	}
	return servers, options
}

// parseBlock рекурсивно разбирает блок конфигурации.
func parseBlock(lines []string, baseDir, sourceFile string) ([]Node, map[string][]string) {
	var directives []Node
	upstreams := make(map[string][]string)
	i := 0

	for i < len(lines) {
		line := stripComments(lines[i])
		if line == "" {
			i++
			continue
		}

		if strings.HasPrefix(line, "include ") {
			pattern := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "include "), ";"))
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(baseDir, pattern)
			}
			matches, _ := filepath.Glob(pattern)
			for _, incPath := range matches {
				incLines, err := readLines(incPath)
				if err != nil {
					continue
				}
				incDir := filepath.Dir(incPath)
				incDirectives, incUpstreams := parseBlock(incLines, incDir, incPath)
				directives = append(directives, incDirectives...)
				for k, v := range incUpstreams {
					upstreams[k] = append(upstreams[k], v...)
				}
			}
			i++
			continue
		}

		if m := reUpstream.FindStringSubmatch(line); m != nil {
			name := m[1]
			blockLines, newI, inline := parseBlockContent(line, lines, i)
			i = newI
			if inline != "" {
				blockLines = append([]string{inline}, blockLines...)
			}
			servers, options := parseUpstreamBlockLines(blockLines)
			upstreams[name] = servers
			directives = append(directives, Node{
				Upstream:   name,
				Servers:    servers,
				Options:    options,
				File:       sourceFile,
			})
			continue
		}

		if strings.HasPrefix(line, "location ") {
			mod, path := parseLocationHeader(line)
			blockLines, newI, inline := parseBlockContent(line, lines, i)
			i = newI
			if inline != "" {
				blockLines = append([]string{inline}, blockLines...)
			}
			subDirectives, subUpstreams := parseBlock(blockLines, baseDir, sourceFile)
			directives = append(directives, Node{
				Block: "location", LocModifier: mod, Arg: path,
				Directives: subDirectives, File: sourceFile, Line: i + 1,
			})
			for k, v := range subUpstreams {
				upstreams[k] = append(upstreams[k], v...)
			}
			continue
		}

		if m := reBlock.FindStringSubmatch(line); m != nil {
			blockName := m[1]
			blockArg := ""
			if len(m) > 2 {
				blockArg = m[2]
			}
			blockLines, newI, inline := parseBlockContent(line, lines, i)
			i = newI
			if inline != "" {
				blockLines = append([]string{inline}, blockLines...)
			}
			subDirectives, subUpstreams := parseBlock(blockLines, baseDir, sourceFile)
			node := Node{
				Block:      blockName,
				Arg:        blockArg,
				Directives: subDirectives,
				File:       sourceFile,
				Line:         i + 1,
			}
			directives = append(directives, node)
			for k, v := range subUpstreams {
				upstreams[k] = append(upstreams[k], v...)
			}
			continue
		}

		if m := reDirective.FindStringSubmatch(line); m != nil {
			directives = append(directives, Node{
				Directive: m[1],
				Args:      m[2],
				File:      sourceFile,
			})
		}
		i++
	}
	return directives, upstreams
}

// parseBlockContent разбирает содержимое блока, включая inline на той же строке.
func parseBlockContent(line string, lines []string, start int) ([]string, int, string) {
	braceIdx := strings.Index(line, "{")
	if braceIdx < 0 {
		bl, ni := collectBlock(lines, start)
		return bl, ni, ""
	}
	afterBrace := line[braceIdx+1:]
	closeIdx := strings.LastIndex(afterBrace, "}")
	if closeIdx >= 0 {
		inline := strings.TrimSpace(afterBrace[:closeIdx])
		return nil, start + 1, inline
	}
	bl, ni := collectBlock(lines, start)
	return bl, ni, ""
}

// collectBlock собирает строки внутри блока { ... }.
func collectBlock(lines []string, start int) ([]string, int) {
	var blockLines []string
	depth := 1
	i := start + 1
	for i < len(lines) && depth > 0 {
		l := stripComments(lines[i])
		depth += strings.Count(l, "{")
		depth -= strings.Count(l, "}")
		if depth > 0 {
			blockLines = append(blockLines, l)
		}
		i++
	}
	return blockLines, i
}

// parseLocationHeader разбирает заголовок location с модификатором.
func parseLocationHeader(line string) (modifier, path string) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "location "))
	if idx := strings.Index(rest, "{"); idx >= 0 {
		rest = strings.TrimSpace(rest[:idx])
	}
	return ParseLocationArg(rest)
}

// readLines читает все строки файла.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
