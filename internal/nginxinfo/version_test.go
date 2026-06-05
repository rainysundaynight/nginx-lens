package nginxinfo

import (
	"testing"
)

const sampleNginxV = `nginx version: nginx/1.24.0
built by gcc 12.2.0 (Debian 12.2.0-14)
built with OpenSSL 3.0.11 19 Sep 2023
TLS SNI support enabled
configure arguments: --prefix=/etc/nginx --sbin-path=/usr/sbin/nginx --modules-path=/usr/lib/nginx/modules --conf-path=/etc/nginx/nginx.conf --error-log-path=/var/log/nginx/error.log --http-log-path=/var/log/nginx/access.log --pid-path=/run/nginx.pid --lock-path=/var/lock/nginx.lock --http-client-body-temp-path=/var/lib/nginx/body --http-fastcgi-temp-path=/var/lib/nginx/fastcgi --http-proxy-temp-path=/var/lib/nginx/proxy --http-scgi-temp-path=/var/lib/nginx/scgi --http-uwsgi-temp-path=/var/lib/nginx/uwsgi --with-compat --with-debug --with-pcre-jit --with-http_ssl_module --with-http_v2_module --with-http_realip_module --with-http_stub_status_module --with-stream --with-stream_ssl_module --add-dynamic-module=/tmp/ngx_brotli
`

func TestParseVersionOutput(t *testing.T) {
	info, err := ParseVersionOutput(sampleNginxV)
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "1.24.0" {
		t.Fatalf("version: %s", info.Version)
	}
	want := map[string]bool{"http_ssl": true, "http_v2": true, "http_realip": true, "stream": true}
	for _, m := range info.StaticModules {
		delete(want, m)
	}
	if len(want) > 0 {
		t.Fatalf("missing modules %v in %v", want, info.StaticModules)
	}
	if len(info.DynamicModules) != 1 || info.DynamicModules[0] != "ngx_brotli" {
		t.Fatalf("dynamic: %v", info.DynamicModules)
	}
}

func TestParseVersionOutputEmpty(t *testing.T) {
	if _, err := ParseVersionOutput(""); err == nil {
		t.Fatal("expected error")
	}
}
