package nginxload

import (
	"path/filepath"
	"runtime"
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

func testCfg(t *testing.T) config.Config {
	t.Helper()
	root := repoRoot(t)
	cfg := config.DefaultConfig()
	cfg.Defaults.NginxConfigPath = filepath.Join(root, "testdata", "nginx", "minimal.conf")
	cfg.Parser.Mode = "regex"
	cfg.Docker.Enabled = "false"
	return cfg
}

func TestParseFile(t *testing.T) {
	cfg := testCfg(t)
	tree, err := ParseFile(cfg, cfg.Defaults.NginxConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.GetUpstreams()) == 0 {
		t.Fatal("expected upstreams")
	}
}

func TestBuildTree(t *testing.T) {
	cfg := testCfg(t)
	tree, path, err := BuildTree(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tree == nil || path == "" {
		t.Fatal("expected tree and path")
	}
}

func TestDockerContext(t *testing.T) {
	cfg := testCfg(t)
	ctx, err := DockerContext(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.HostConfigPath != cfg.Defaults.NginxConfigPath {
		t.Fatalf("host path=%q", ctx.HostConfigPath)
	}
}
