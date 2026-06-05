package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleUpstream(t *testing.T) {
	conf := `
upstream backend {
    server 127.0.0.1:8080;
    server 10.0.0.1:80;
}
`
	f := writeTemp(t, conf)
	tree, err := ParseNginxConfig(f)
	require.NoError(t, err)
	ups := tree.GetUpstreams()
	assert.Contains(t, ups, "backend")
	assert.ElementsMatch(t, []string{"127.0.0.1:8080", "10.0.0.1:80"}, ups["backend"])
}

func TestUpstreamWithInclude(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "nginx.conf")
	subPath := filepath.Join(dir, "sub.conf")
	require.NoError(t, os.WriteFile(mainPath, []byte("include sub.conf;\n"), 0644))
	require.NoError(t, os.WriteFile(subPath, []byte(`
upstream api {
    server api1:9000;
}
`), 0644))
	tree, err := ParseNginxConfig(mainPath)
	require.NoError(t, err)
	ups := tree.GetUpstreams()
	assert.Contains(t, ups, "api")
	assert.Equal(t, []string{"api1:9000"}, ups["api"])
}

func TestNestedBlocksAndComments(t *testing.T) {
	conf := `
# http block
http {
    upstream u1 { server 1.1.1.1:80; }
    server {
        listen 80;
        location /api {
            proxy_pass http://u1;
        }
    }
}
`
	f := writeTemp(t, conf)
	tree, err := ParseNginxConfig(f)
	require.NoError(t, err)
	ups := tree.GetUpstreams()
	assert.Contains(t, ups, "u1")
	assert.Equal(t, []string{"1.1.1.1:80"}, ups["u1"])
}

func TestLocationModifierParsing(t *testing.T) {
	conf := `server {
    location = /api { return 200; }
    location ^~ /static/ { root /var/www; }
}`
	f := writeTemp(t, conf)
	tree, err := ParseNginxConfig(f)
	require.NoError(t, err)
	var exact, prefix *Node
	for _, n := range tree.Directives[0].Directives {
		if n.Block != "location" {
			continue
		}
		if n.LocModifier == "=" {
			exact = &n
		}
		if n.LocModifier == "^~" {
			prefix = &n
		}
	}
	require.NotNil(t, exact)
	assert.Equal(t, "/api", exact.Arg)
	require.NotNil(t, prefix)
	assert.Equal(t, "/static/", prefix.Arg)
}

func TestUpstreamBlockOptionsPreserved(t *testing.T) {
	conf := `
upstream back {
    least_conn;
    keepalive 32;
    keepalive_timeout 60s;
    server 127.0.0.1:8080 weight=3 max_fails=2 fail_timeout=20s;
    server 127.0.0.1:8081 backup;
}
`
	f := writeTemp(t, conf)
	tree, err := ParseNginxConfig(f)
	require.NoError(t, err)
	var ublock *Node
	for i := range tree.Directives {
		if tree.Directives[i].Upstream == "back" {
			ublock = &tree.Directives[i]
			break
		}
	}
	require.NotNil(t, ublock)
	var dirs []string
	for _, o := range ublock.Options {
		dirs = append(dirs, o.Name)
	}
	assert.Contains(t, dirs, "least_conn")
	assert.NotEmpty(t, ublock.Options)
	assert.Len(t, ublock.Servers, 2)
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "nginx-*.conf")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
