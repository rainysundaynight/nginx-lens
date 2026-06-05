package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/config"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func setupCLITest(t *testing.T) {
	t.Helper()
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NGINX_LENS_CONFIG", filepath.Join(root, "testdata", "config", "cli-test.yaml"))
	config.Reload()
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	setupCLITest(t)
	return executeCLI(args...)
}

func runCLIWithConfig(t *testing.T, configPath string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NGINX_LENS_CONFIG", configPath)
	config.Reload()
	return executeCLI(args...)
}

func executeCLI(args ...string) (stdout, stderr string, err error) {
	outR, outW, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	cmd := NewRoot()
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	var outBuf, errBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, outR)
	_, _ = io.Copy(&errBuf, errR)
	return outBuf.String(), errBuf.String(), runErr
}

func TestCLIVersion(t *testing.T) {
	out, _, err := runCLI(t, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("version output empty")
	}
}

func TestCLIConfigValidate(t *testing.T) {
	out, _, err := runCLI(t, "config", "validate")
	if err != nil {
		t.Fatalf("config validate: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "корректна") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestCLIAnalyzeJSON(t *testing.T) {
	out, _, err := runCLI(t, "analyze")
	if err != nil {
		t.Fatalf("analyze: %v\nout: %s", err, out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid json: %v\nout: %s", err, out)
	}
	if _, ok := result["issues"]; !ok {
		t.Fatalf("missing issues key: %v", result)
	}
	if _, ok := result["summary"]; !ok {
		t.Fatalf("missing summary key: %v", result)
	}
}

func TestCLIRoute(t *testing.T) {
	out, _, err := runCLI(t, "route", "--url", "http://example.com/api")
	if err != nil {
		t.Fatalf("route: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "example.com") && !strings.Contains(out, "location") {
		t.Fatalf("unexpected route output: %q", out)
	}
}

func TestCLIExplain(t *testing.T) {
	out, _, err := runCLI(t, "explain", "--url", "http://example.com/api")
	if err != nil {
		t.Fatalf("explain: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "trace") && !strings.Contains(out, "Trace") {
		var raw map[string]interface{}
		if json.Unmarshal([]byte(out), &raw) == nil {
			if _, ok := raw["trace"]; !ok {
				t.Fatalf("missing trace in json: %q", out)
			}
		} else if !strings.Contains(strings.ToLower(out), "trace") {
			t.Fatalf("unexpected explain output: %q", out)
		}
	}
}

func TestCLIScoreJSON(t *testing.T) {
	out, _, err := runCLI(t, "score")
	if err != nil {
		t.Fatalf("score: %v\nout: %s", err, out)
	}
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid score json: %v\nout: %s", err, out)
	}
	if _, ok := report["total"]; !ok {
		t.Fatalf("missing total in score: %v", report)
	}
}

func TestCLIDiffJSON(t *testing.T) {
	out, _, err := runCLI(t, "diff")
	if err != nil {
		t.Fatalf("diff: %v\nout: %s", err, out)
	}
	var diffs []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &diffs); err != nil {
		t.Fatalf("invalid diff json: %v\nout: %s", err, out)
	}
	if len(diffs) == 0 {
		t.Fatal("diff должен найти изменения upstream")
	}
}

func TestCLIValidateJSON(t *testing.T) {
	out, _, err := runCLI(t, "validate")
	if err != nil {
		t.Fatalf("validate: %v\nout: %s", err, out)
	}
	idx := strings.Index(out, "{")
	if idx < 0 {
		t.Fatalf("missing json in output: %q", out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out[idx:]), &result); err != nil {
		t.Fatalf("invalid validate json: %v\nout: %s", err, out)
	}
	if _, ok := result["analysis"]; !ok {
		t.Fatalf("missing analysis key: %v", result)
	}
}

func TestCLIBlastRadius(t *testing.T) {
	out, _, err := runCLI(t, "blast-radius", "api_backend")
	if err != nil {
		t.Fatalf("blast-radius: %v\nout: %s", err, out)
	}
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid blast-radius json: %v\nout: %s", err, out)
	}
	if _, ok := report["upstream"]; !ok {
		t.Fatalf("missing upstream key: %v", report)
	}
}

func TestCLITreeMarkdown(t *testing.T) {
	out, _, err := runCLI(t, "tree")
	if err != nil {
		t.Fatalf("tree: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "server") {
		t.Fatalf("unexpected tree output: %q", out)
	}
}

func TestCLIGraph(t *testing.T) {
	out, _, err := runCLI(t, "graph")
	if err != nil {
		t.Fatalf("graph: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "LOCATION") && !strings.Contains(out, "Route graph") {
		t.Fatalf("unexpected graph output: %q", out)
	}
}

func TestCLIUpstreamsJSON(t *testing.T) {
	out, _, err := runCLI(t, "upstreams")
	if err != nil {
		t.Fatalf("upstreams: %v\nout: %s", err, out)
	}
	var blocks []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &blocks); err != nil {
		t.Fatalf("invalid upstreams json: %v\nout: %s", err, out)
	}
	if len(blocks) == 0 {
		t.Fatal("ожидались upstream blocks")
	}
}

func TestCLIIngressAuditJSON(t *testing.T) {
	out, _, err := runCLI(t, "ingress-audit")
	if err != nil {
		t.Fatalf("ingress-audit: %v\nout: %s", err, out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid ingress-audit json: %v\nout: %s", err, out)
	}
	if _, ok := result["issues"]; !ok {
		t.Fatalf("missing issues key: %v", result)
	}
}

func TestCLIConfigShow(t *testing.T) {
	out, _, err := runCLI(t, "config")
	if err != nil {
		t.Fatalf("config: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "Config path:") {
		t.Fatalf("unexpected config output: %q", out)
	}
}

func TestCLIResolveJSON(t *testing.T) {
	out, _, err := runCLI(t, "resolve")
	if err != nil {
		t.Fatalf("resolve: %v\nout: %s", err, out)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid resolve json: %v\nout: %s", err, out)
	}
	if _, ok := result["upstreams"]; !ok {
		t.Fatalf("missing upstreams key: %v", result)
	}
}

func TestCLIMetricsJSON(t *testing.T) {
	out, _, err := runCLI(t, "metrics")
	if err != nil {
		t.Fatalf("metrics: %v\nout: %s", err, out)
	}
	var metrics map[string]interface{}
	if err := json.Unmarshal([]byte(out), &metrics); err != nil {
		t.Fatalf("invalid metrics json: %v\nout: %s", err, out)
	}
	if _, ok := metrics["servers"]; !ok {
		t.Fatalf("missing servers key: %v", metrics)
	}
}

func TestCLIIncludeTreeJSON(t *testing.T) {
	out, _, err := runCLI(t, "include-tree")
	if err != nil {
		t.Fatalf("include-tree: %v\nout: %s", err, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "[") && !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("unexpected include-tree output: %q", out)
	}
}

func TestCLIAnalyzeTable(t *testing.T) {
	root := repoRoot(t)
	cfgPath := writeTempCLIConfig(t, root, "table")
	out, _, err := runCLIWithConfig(t, cfgPath, "analyze")
	if err != nil {
		t.Fatalf("analyze table: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "TYPE") && !strings.Contains(out, "Проблем не найдено") {
		t.Fatalf("unexpected table output: %q", out)
	}
}

func writeTempCLIConfig(t *testing.T, root, format string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "defaults:\n  nginx_config_path: " + filepath.ToSlash(filepath.Join(root, "testdata", "nginx", "minimal.conf")) + "\n  timeout: 3.0\ndocker:\n  enabled: false\nparser:\n  mode: regex\noutput:\n  colors: true\n  format: " + format + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
