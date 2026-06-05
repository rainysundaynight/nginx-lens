package parser

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ---------- Парсер nginx -T ----------
// Разбор expanded-конфигурации с маркерами файлов.

var reConfigFileMarker = regexp.MustCompile(`^#\s*configuration file\s+(.+):\s*$`)

// ParseExpanded запускает nginx -T и парсит вывод.
func ParseExpanded(configPath, nginxPath string) (*ConfigTree, error) {
	if nginxPath == "" {
		nginxPath = "nginx"
	}
	cmd := exec.Command(nginxPath, "-T", "-c", configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("nginx -T: %w: %s", err, string(out))
	}
	return ParseExpandedOutput(string(out))
}

// ParseExpandedOutput парсит вывод nginx -T.
func ParseExpandedOutput(output string) (*ConfigTree, error) {
	sections := splitExpandedSections(output)
	expandedOpts := parseBlockOptions{skipIncludes: true}
	if len(sections) == 0 {
		lines := strings.Split(output, "\n")
		directives, upstreams := parseBlockOpts(lines, "", "", expandedOpts)
		return NewConfigTree(directives, upstreams), nil
	}
	var allDirectives []Node
	allUpstreams := make(map[string][]string)
	for file, lines := range sections {
		directives, upstreams := parseBlockOpts(lines, "", file, expandedOpts)
		allDirectives = append(allDirectives, directives...)
		for k, v := range upstreams {
			allUpstreams[k] = append(allUpstreams[k], v...)
		}
	}
	return NewConfigTree(allDirectives, DedupeUpstreamMap(allUpstreams)), nil
}

func splitExpandedSections(output string) map[string][]string {
	sections := make(map[string][]string)
	var currentFile string
	var currentLines []string
	for _, line := range strings.Split(output, "\n") {
		if m := reConfigFileMarker.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			if currentFile != "" {
				sections[currentFile] = currentLines
			}
			currentFile = strings.TrimSpace(m[1])
			currentLines = nil
			continue
		}
		if currentFile != "" {
			currentLines = append(currentLines, line)
		}
	}
	if currentFile != "" {
		sections[currentFile] = currentLines
	}
	return sections
}
