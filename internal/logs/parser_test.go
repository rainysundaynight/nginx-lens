package logs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLogFileCombined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	line := `127.0.0.1 - - [05/Jun/2026:10:00:00 +0000] "GET /api HTTP/1.1" 200 123 "-" "curl/8.0"`
	if err := os.WriteFile(path, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stats, err := ParseLogFile(path, 5, "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalRequests != 1 {
		t.Fatalf("requests=%d", stats.TotalRequests)
	}
	if stats.StatusCodes["200"] != 1 {
		t.Fatalf("status codes=%v", stats.StatusCodes)
	}
}

func TestParseLineCombined(t *testing.T) {
	line := `127.0.0.1 - - [05/Jun/2026:10:00:00 +0000] "GET /health HTTP/1.1" 200 10 "-" "test"`
	parsed := parseLine(line)
	if parsed == nil || parsed.Path != "/health" || parsed.Status != 200 {
		t.Fatalf("parsed=%v", parsed)
	}
}
