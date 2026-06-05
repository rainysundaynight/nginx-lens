package docker

import (
	"path/filepath"
	"testing"
)

func TestMapToHost(t *testing.T) {
	m := map[string]string{"/etc/nginx": "/opt/nginx/conf"}
	got, ok := MapToHost(m, "/etc/nginx/nginx.conf")
	want := filepath.Join("/opt/nginx/conf", "nginx.conf")
	if !ok || got != want {
		t.Fatalf("map failed: got %q want %q ok=%v", got, want, ok)
	}
}

func TestBuildContextHostFile(t *testing.T) {
	cfg := Config{Enabled: "false"}
	ctx, err := BuildContext(cfg, "/etc/hosts")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.UseExec {
		t.Fatal("expected host mode")
	}
}
