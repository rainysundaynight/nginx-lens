import typer
from typer.testing import CliRunner
from commands.cli import app
import pytest

runner = CliRunner()

def test_resolve_basic(monkeypatch):
    """Тест базовой функциональности resolve"""
    from commands import resolve as resolve_mod
    monkeypatch.setattr(resolve_mod, "parse_nginx_config", lambda path: type("T", (), {"get_upstreams": lambda self: {"test_up": ["example.com:80", "127.0.0.1:8080"]}})())
    monkeypatch.setattr(resolve_mod, "resolve_upstreams", lambda ups, max_workers: {
        "test_up": [
            {"address": "example.com:80", "resolved": ["93.184.216.34:80"]},
            {"address": "127.0.0.1:8080", "resolved": ["127.0.0.1:8080"]}
        ]
    })
    result = runner.invoke(app, ["resolve", "nginx.conf"])
    assert result.exit_code == 0
    assert "test_up" in result.output
    assert "example.com:80" in result.output
    assert "127.0.0.1:8080" in result.output

def test_resolve_failed(monkeypatch):
    """Тест resolve с невалидным DNS"""
    from commands import resolve as resolve_mod
    monkeypatch.setattr(resolve_mod, "parse_nginx_config", lambda path: type("T", (), {"get_upstreams": lambda self: {"test_up": ["nonexistent-domain-12345.invalid:80"]}})())
    monkeypatch.setattr(resolve_mod, "resolve_upstreams", lambda ups, max_workers: {
        "test_up": [
            {"address": "nonexistent-domain-12345.invalid:80", "resolved": []}
        ]
    })
    result = runner.invoke(app, ["resolve", "nginx.conf"])
    assert result.exit_code == 1
    assert "Failed to resolve" in result.output

def test_resolve_invalid_cname(monkeypatch):
    """Тест resolve с невалидным CNAME"""
    from commands import resolve as resolve_mod
    monkeypatch.setattr(resolve_mod, "parse_nginx_config", lambda path: type("T", (), {"get_upstreams": lambda self: {"test_up": ["example.com:80"]}})())
    monkeypatch.setattr(resolve_mod, "resolve_upstreams", lambda ups, max_workers: {
        "test_up": [
            {"address": "example.com:80", "resolved": ["invalid resolve (via cname.example.com -> TXT)"]}
        ]
    })
    result = runner.invoke(app, ["resolve", "nginx.conf"])
    assert result.exit_code == 1
    assert "invalid resolve" in result.output

def test_resolve_no_upstreams(monkeypatch):
    """Тест resolve когда нет upstream"""
    from commands import resolve as resolve_mod
    monkeypatch.setattr(resolve_mod, "parse_nginx_config", lambda path: type("T", (), {"get_upstreams": lambda self: {}})())
    result = runner.invoke(app, ["resolve", "nginx.conf"])
    assert result.exit_code == 0
    assert "не найдено" in result.output.lower() or "not found" in result.output.lower()

def test_resolve_max_workers_option(monkeypatch):
    """Тест опции --max-workers"""
    from commands import resolve as resolve_mod
    monkeypatch.setattr(resolve_mod, "parse_nginx_config", lambda path: type("T", (), {"get_upstreams": lambda self: {"test_up": ["example.com:80"]}})())
    monkeypatch.setattr(resolve_mod, "resolve_upstreams", lambda ups, max_workers: {
        "test_up": [{"address": "example.com:80", "resolved": ["93.184.216.34:80"]}]
    })
    result = runner.invoke(app, ["resolve", "nginx.conf", "--max-workers", "20"])
    assert result.exit_code == 0

