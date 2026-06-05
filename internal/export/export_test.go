package export

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
)

func TestFormatAnalyzeResults(t *testing.T) {
	result := analyzer.AnalysisResult{
		EmptyBlocks: []analyzer.EmptyBlock{
			{Block: "location", Arg: "/empty", File: "test.conf", Line: 1},
		},
	}
	out := FormatAnalyzeResults(result, analyzer.FilterOptions{})
	if len(out.Issues) == 0 {
		t.Fatal("ожидались issues")
	}
	if out.Summary["low"] != 1 {
		t.Fatalf("summary low=%d, want 1", out.Summary["low"])
	}
}

func TestWriteToJSONYAML(t *testing.T) {
	v := map[string]string{"key": "value"}
	var buf bytes.Buffer
	if err := WriteTo(&buf, "json", v); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"key"`) {
		t.Fatalf("unexpected json: %s", buf.String())
	}
	buf.Reset()
	if err := WriteTo(&buf, "yaml", v); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "key:") {
		t.Fatalf("unexpected yaml: %s", buf.String())
	}
}

func TestWriteToUnknownFormat(t *testing.T) {
	err := WriteTo(&bytes.Buffer{}, "xml", nil)
	if err == nil {
		t.Fatal("ожидалась ошибка для неизвестного формата")
	}
}

func TestFormatPrometheus(t *testing.T) {
	out := FormatPrometheus(
		map[string]float64{
			"nginx_lens_score":            85,
			"nginx_lens_upstream_healthy": 1,
		},
		map[string]map[string]string{
			"nginx_lens_upstream_healthy": {"upstream": "api", "address": "127.0.0.1:8080"},
		},
	)
	if !strings.Contains(out, "nginx_lens_score 85") {
		t.Fatalf("missing score metric: %q", out)
	}
	if !strings.Contains(out, `upstream="api"`) {
		t.Fatalf("missing labels: %q", out)
	}
}

func TestWriteCSV(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	err = WriteCSV([]string{"a", "b"}, [][]string{{"1", "2"}})
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "1,2") {
		t.Fatalf("unexpected csv: %q", out)
	}
}

func TestAnalyzeExportJSONRoundtrip(t *testing.T) {
	data := AnalyzeExport{
		Issues:  []analyzer.Issue{{Type: "x", Severity: analyzer.SeverityHigh}},
		Summary: map[string]int{"high": 1},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AnalyzeExport
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Issues) != 1 {
		t.Fatal("roundtrip failed")
	}
}
