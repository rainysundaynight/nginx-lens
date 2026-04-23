import tempfile
import os
from parser.nginx_parser import parse_nginx_config

def test_simple_upstream():
    conf = """
    upstream backend {
        server 127.0.0.1:8080;
        server 10.0.0.1:80;
    }
    """
    with tempfile.NamedTemporaryFile("w+", delete=False) as f:
        f.write(conf)
        f.flush()
        tree = parse_nginx_config(f.name)
    ups = tree.get_upstreams()
    assert "backend" in ups
    assert set(ups["backend"]) == {"127.0.0.1:8080", "10.0.0.1:80"}
    os.unlink(f.name)

def test_upstream_with_include():
    conf_main = """
    include sub.conf;
    """
    conf_sub = """
    upstream api {
        server api1:9000;
    }
    """
    with tempfile.TemporaryDirectory() as d:
        main_path = os.path.join(d, "nginx.conf")
        sub_path = os.path.join(d, "sub.conf")
        with open(main_path, "w") as f:
            f.write(conf_main)
        with open(sub_path, "w") as f:
            f.write(conf_sub)
        tree = parse_nginx_config(main_path)
        ups = tree.get_upstreams()
        assert "api" in ups
        assert ups["api"] == ["api1:9000"]

def test_nested_blocks_and_comments():
    conf = """
    # http block
    http {
        upstream u1 { server 1.1.1.1:80; } # inline
        server { # comment
            listen 80;
            location /api {
                proxy_pass http://u1;
            }
        }
    }
    """
    with tempfile.NamedTemporaryFile("w+", delete=False) as f:
        f.write(conf)
        f.flush()
        tree = parse_nginx_config(f.name)
    ups = tree.get_upstreams()
    assert "u1" in ups
    assert ups["u1"] == ["1.1.1.1:80"]
    os.unlink(f.name)


def test_upstream_block_options_preserved():
    conf = """
    upstream back {
        least_conn;
        keepalive 32;
        keepalive_timeout 60s;
        server 127.0.0.1:8080 weight=3 max_fails=2 fail_timeout=20s;
        server 127.0.0.1:8081 backup;
    }
    """
    with tempfile.NamedTemporaryFile("w+", delete=False) as f:
        f.write(conf)
        f.flush()
        tree = parse_nginx_config(f.name)
    os.unlink(f.name)
    ublocks = [d for d in tree.directives if d.get("upstream") == "back"]
    assert len(ublocks) == 1
    b = ublocks[0]
    assert "least_conn" in [o["directive"] for o in b.get("options", [])]
    assert b.get("options")  # not empty
    assert any("keepalive" in o["directive"] for o in b["options"])
    assert b["servers"], "две server-строки"