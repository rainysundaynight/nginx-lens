import typer
from typer.testing import CliRunner
from commands.cli import app
import pytest
import tempfile
import os

runner = CliRunner()

def test_route_basic(monkeypatch):
    """Тест базовой функциональности route"""
    from commands import route as route_mod
    
    # Создаем временный конфиг файл
    import tempfile
    config_content = """
server {
    server_name example.com;
    listen 80;
    location /api {
        proxy_pass http://backend;
    }
}
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f:
        f.write(config_content)
        f.flush()
        config_path = f.name
    
    try:
        # Мокаем parse_nginx_config чтобы вернуть правильную структуру
        mock_tree = type("T", (), {
            "directives": [
                {
                    "block": "server",
                    "arg": None,
                    "__file__": config_path,
                    "directives": [
                        {"directive": "server_name", "args": "example.com"},
                        {"directive": "listen", "args": "80"},
                        {
                            "block": "location",
                            "arg": "/api",
                            "__file__": config_path,
                            "directives": [
                                {"directive": "proxy_pass", "args": "http://backend"}
                            ]
                        }
                    ]
                }
            ]
        })()
        
        monkeypatch.setattr(route_mod, "parse_nginx_config", lambda path: mock_tree)
        monkeypatch.setattr(route_mod, "find_route", lambda tree, url: {
            "server": {"arg": None, "__file__": config_path},
            "location": {"arg": "/api", "__file__": config_path},
            "proxy_pass": "http://backend"
        })
        
        result = runner.invoke(app, ["route", "http://example.com/api/test", "--config", config_path])
        assert result.exit_code == 0
    finally:
        import os
        os.unlink(config_path)

def test_route_with_config_option(monkeypatch):
    """Тест route с опцией --config"""
    from commands import route as route_mod
    monkeypatch.setattr(route_mod, "parse_nginx_config", lambda path: type("T", (), {
        "directives": []
    })())
    monkeypatch.setattr(route_mod, "find_route", lambda tree, url: None)
    
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f:
        f.write("server { listen 80; }")
        f.flush()
        config_path = f.name
    
    try:
        result = runner.invoke(app, ["route", "http://example.com/test", "--config", config_path])
        assert result.exit_code == 0
    finally:
        os.unlink(config_path)

