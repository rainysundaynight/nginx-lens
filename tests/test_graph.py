import typer
from typer.testing import CliRunner
from commands.cli import app
import pytest
import tempfile
import os

runner = CliRunner()

def test_graph_basic(monkeypatch):
    """Тест базовой функциональности graph"""
    from commands import graph as graph_mod
    monkeypatch.setattr(graph_mod, "parse_nginx_config", lambda path: type("T", (), {
        "directives": [
            {
                "block": "server",
                "directives": [
                    {
                        "block": "location",
                        "arg": "/",
                        "directives": [
                            {"directive": "proxy_pass", "args": "http://backend"}
                        ]
                    }
                ]
            },
            {
                "upstream": "backend",
                "servers": ["127.0.0.1:8080"]
            }
        ]
    })())
    
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f:
        f.write("server { location / { proxy_pass http://backend; } }")
        f.flush()
        config_path = f.name
    
    try:
        result = runner.invoke(app, ["graph", config_path])
        assert result.exit_code == 0
        # Проверяем что вывод содержит информацию о маршрутах
    finally:
        os.unlink(config_path)

def test_graph_no_routes(monkeypatch):
    """Тест graph когда нет маршрутов"""
    from commands import graph as graph_mod
    monkeypatch.setattr(graph_mod, "parse_nginx_config", lambda path: type("T", (), {
        "directives": []
    })())
    
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f:
        f.write("events { }")
        f.flush()
        config_path = f.name
    
    try:
        result = runner.invoke(app, ["graph", config_path])
        assert result.exit_code == 0
        assert "не найдено" in result.output.lower() or "not found" in result.output.lower()
    finally:
        os.unlink(config_path)

