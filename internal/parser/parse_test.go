package parser

import (
	"os"
	"testing"
)

func TestParseLocationArg(t *testing.T) {
	cases := []struct {
		raw, mod, path string
	}{
		{"/api", "", "/api"},
		{"= /api", "=", "/api"},
		{"^~ /static", "^~", "/static"},
		{"~ \\.php$", "~", "\\.php$"},
		{"~* \\.jpg", "~*", "\\.jpg"},
	}
	for _, c := range cases {
		mod, path := ParseLocationArg(c.raw)
		if mod != c.mod || path != c.path {
			t.Errorf("ParseLocationArg(%q) = %q,%q; want %q,%q", c.raw, mod, path, c.mod, c.path)
		}
	}
}

func TestParseConfigRegexMode(t *testing.T) {
	tree, err := ParseConfig(writeTempParse(t, `upstream x { server 1.2.3.4:80; }`), ParseOptions{Mode: "regex"})
	if err != nil {
		t.Fatal(err)
	}
	if tree == nil || len(tree.GetUpstreams()) == 0 {
		t.Fatal("ожидалось непустое дерево")
	}
}

func writeTempParse(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "nginx-*.conf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
