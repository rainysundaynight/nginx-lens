package analyzer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

func TestAuditCertificatesExpired(t *testing.T) {
	certPath := writeExpiredCert(t)
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		Directives: []parser.Node{
			{Directive: "server_name", Args: "test.local"},
			{Directive: "ssl_certificate", Args: certPath},
		},
	}}, nil)
	issues := AuditCertificates(tree, 30, nil)
	found := false
	for _, iss := range issues {
		if iss.Type == "cert_expired" {
			found = true
		}
	}
	if !found {
		t.Fatal("ожидался cert_expired")
	}
}

func TestAuditCertificatesReadCustom(t *testing.T) {
	pemData := []byte("not-a-cert")
	tree := parser.NewConfigTree([]parser.Node{{
		Block: "server",
		File:  "/etc/nginx/conf.d/a.conf",
		Directives: []parser.Node{
			{Directive: "server_name", Args: "a.example", File: "/etc/nginx/conf.d/a.conf"},
			{Directive: "ssl_certificate", Args: "/etc/nginx/ssl/a.pem", File: "/etc/nginx/conf.d/a.conf"},
		},
	}}, nil)
	called := false
	issues := AuditCertificatesRead(tree, 30, func(path string) ([]byte, error) {
		called = true
		if path != "/etc/nginx/ssl/a.pem" {
			t.Fatalf("path=%q", path)
		}
		return pemData, nil
	})
	if !called {
		t.Fatal("reader не вызван")
	}
	found := false
	for _, iss := range issues {
		if iss.Type == "cert_invalid_pem" {
			found = true
		}
	}
	if !found {
		t.Fatal("ожидался cert_invalid_pem от docker/custom reader")
	}
}

func TestAuditCertificatesFromConfDRelative(t *testing.T) {
	dir := t.TempDir()
	confd := filepath.Join(dir, "conf.d")
	ssld := filepath.Join(dir, "ssl")
	if err := os.MkdirAll(confd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ssld, 0o755); err != nil {
		t.Fatal(err)
	}
	certPath := writeExpiredCertAt(t, filepath.Join(ssld, "site.pem"))
	main := filepath.Join(dir, "nginx.conf")
	site := filepath.Join(confd, "site.conf")
	if err := os.WriteFile(main, []byte("http {\n  include conf.d/*.conf;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	siteBody := "server {\n  listen 443 ssl;\n  server_name site.local;\n  ssl_certificate ../ssl/site.pem;\n}\n"
	if err := os.WriteFile(site, []byte(siteBody), 0o644); err != nil {
		t.Fatal(err)
	}
	tree, err := parser.ParseNginxConfig(main)
	if err != nil {
		t.Fatal(err)
	}
	issues := AuditCertificates(tree, 30, nil)
	found := false
	for _, iss := range issues {
		if iss.Type == "cert_expired" && filepath.Clean(iss.CertPath) == filepath.Clean(certPath) {
			found = true
		}
	}
	if !found {
		t.Fatalf("ожидался cert_expired для %s, issues=%+v", certPath, issues)
	}
}

func TestResolveCertPath(t *testing.T) {
	got := resolveCertPath("../ssl/a.pem", "/etc/nginx/conf.d/x.conf")
	want := filepath.Clean("/etc/nginx/ssl/a.pem")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if resolveCertPath("/abs/a.pem", "/etc/nginx/conf.d/x.conf") != filepath.Clean("/abs/a.pem") {
		t.Fatal("absolute path must stay")
	}
}

func writeExpiredCert(t *testing.T) string {
	t.Helper()
	return writeExpiredCertAt(t, filepath.Join(t.TempDir(), "expired.pem"))
}

func writeExpiredCertAt(t *testing.T, path string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test.local"},
		NotBefore:    time.Now().Add(-48 * time.Hour),
		NotAfter:     time.Now().Add(-24 * time.Hour),
		DNSNames:     []string{"test.local"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	_ = f.Close()
	return path
}
