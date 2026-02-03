import typer
from typer.testing import CliRunner
from commands.cli import app
import pytest

runner = CliRunner()

def test_health(monkeypatch):
    # Мокаем парсер и чекер
    from commands import health as health_mod
    monkeypatch.setattr(health_mod, "parse_nginx_config", lambda path: type("T", (), {"get_upstreams": lambda self: {"test_up": ["127.0.0.1:9999", "badhost:80"]}})())
    monkeypatch.setattr(health_mod, "check_upstreams", lambda ups, timeout, retries, mode, max_workers: {"test_up": [{"address": "127.0.0.1:9999", "healthy": True}, {"address": "badhost:80", "healthy": False}]})
    result = runner.invoke(app, ["health", "nginx.conf"])
    assert "test_up" in result.output or "127.0.0.1:9999" in result.output
    assert "127.0.0.1:9999" in result.output
    assert "Healthy" in result.output
    assert "badhost:80" in result.output
    assert "Unhealthy" in result.output
    assert result.exit_code == 1  # Должен быть 1 из-за unhealthy upstream 