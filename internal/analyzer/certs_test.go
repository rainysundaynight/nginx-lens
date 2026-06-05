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

func writeExpiredCert(t *testing.T) string {
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
	path := filepath.Join(t.TempDir(), "expired.pem")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	f.Close()
	return path
}
