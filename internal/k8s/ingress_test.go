package k8s

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestAuditIngressManifests(t *testing.T) {
	root := repoRoot(t)
	manifestsPath := filepath.Join(root, "testdata", "k8s")
	names := CollectServerNames([]string{"example.com"})
	issues, rules := AuditIngressManifests(manifestsPath, names)
	if len(rules) == 0 {
		t.Fatal("ожидались ingress rules")
	}
	foundOrphan := false
	foundNoTLS := false
	for _, iss := range issues {
		switch iss.Type {
		case "orphaned_ingress":
			if iss.Host == "unknown.example.com" {
				foundOrphan = true
			}
		case "ingress_no_tls":
			foundNoTLS = true
		}
	}
	if !foundOrphan {
		t.Fatal("ожидался orphaned_ingress для unknown.example.com")
	}
	if !foundNoTLS {
		t.Fatal("ожидался ingress_no_tls")
	}
}

func TestCollectServerNames(t *testing.T) {
	m := CollectServerNames([]string{"a.com", "b.com"})
	if len(m) != 2 {
		t.Fatalf("len=%d, want 2", len(m))
	}
	if _, ok := m["a.com"]; !ok {
		t.Fatal("missing a.com")
	}
}
