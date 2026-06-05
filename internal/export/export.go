package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
	"gopkg.in/yaml.v3"
)

// ---------- Экспорт результатов ----------
// JSON, YAML, CSV и Prometheus форматы.

// PrintJSON выводит данные в JSON.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintYAML выводит данные в YAML.
func PrintYAML(v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

// AnalyzeExport — структура экспорта analyze.
type AnalyzeExport struct {
	Issues  []analyzer.Issue `json:"issues"`
	Summary map[string]int   `json:"summary"`
}

// FormatAnalyzeResults форматирует результаты анализа для экспорта.
func FormatAnalyzeResults(result analyzer.AnalysisResult, filter analyzer.FilterOptions) AnalyzeExport {
	issues := analyzer.CollectIssues(result)
	var filtered []analyzer.Issue
	summary := map[string]int{"high": 0, "medium": 0, "low": 0}
	for _, issue := range issues {
		if analyzer.ShouldIncludeIssue(issue.Type, issue.Severity, filter) {
			filtered = append(filtered, issue)
			summary[string(issue.Severity)]++
		}
	}
	return AnalyzeExport{Issues: filtered, Summary: summary}
}

// AppendIssue добавляет issue в экспорт с учётом фильтра и summary.
func AppendIssue(exp *AnalyzeExport, issue analyzer.Issue, filter analyzer.FilterOptions) {
	if !analyzer.ShouldIncludeIssue(issue.Type, issue.Severity, filter) {
		return
	}
	exp.Issues = append(exp.Issues, issue)
	if exp.Summary == nil {
		exp.Summary = map[string]int{"high": 0, "medium": 0, "low": 0}
	}
	exp.Summary[string(issue.Severity)]++
}

// HealthExport — экспорт health.
type HealthExport struct {
	Upstreams map[string][]upstream.ServerHealth `json:"upstreams"`
}

// ResolveExport — экспорт resolve.
type ResolveExport struct {
	Upstreams map[string][]upstream.ServerResolve `json:"upstreams"`
}

// WriteCSV записывает строки в CSV.
func WriteCSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// FormatPrometheus форматирует метрики в Prometheus text format.
func FormatPrometheus(metrics map[string]float64, labels map[string]map[string]string) string {
	var b strings.Builder
	for name, value := range metrics {
		if lbls, ok := labels[name]; ok && len(lbls) > 0 {
			var parts []string
			for k, v := range lbls {
				parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
			}
			b.WriteString(fmt.Sprintf("%s{%s} %g\n", name, strings.Join(parts, ","), value))
		} else {
			b.WriteString(fmt.Sprintf("%s %g\n", name, value))
		}
	}
	return b.String()
}

// WriteTo записывает данные в writer.
func WriteTo(w io.Writer, format string, v interface{}) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case "yaml":
		data, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}
