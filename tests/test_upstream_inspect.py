import os
import tempfile

from parser.nginx_parser import parse_nginx_config
from utils.upstream_inspect import (
    collect_known_upstream_names,
    find_upstream_references,
    parse_server_options,
)


def test_parse_server_line_weight():
    p = parse_server_options("10.0.0.1:80 weight=5 backup")
    assert p["address"] == "10.0.0.1:80"
    assert p["weight"] == "5"
    assert p["backup"] == "да"
    assert p["down"] == "—"


def test_find_references_http_proxy():
    conf = """
    http {
        upstream api {
            server 127.0.0.1:9000;
        }
        server {
            listen 80;
            server_name ex.com;
            location /p {
                proxy_pass http://api/;
            }
        }
    }
    """
    with tempfile.NamedTemporaryFile("w+", delete=False) as f:
        f.write(conf)
        f.flush()
        p = f.name
    try:
        tree = parse_nginx_config(p)
    finally:
        os.unlink(p)
    known = collect_known_upstream_names(tree.directives)
    assert "api" in known
    refs = find_upstream_references(tree.directives, known)
    assert any(r.upstream_name == "api" and r.from_directive == "proxy_pass" for r in refs)
    r0 = [r for r in refs if r.upstream_name == "api"][0]
    assert "ex.com" in r0.server_name
    assert "80" in r0.listen
    assert r0.location == "/p"
    assert r0.is_stream is False
