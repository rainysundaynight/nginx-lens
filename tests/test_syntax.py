"""Тесты команды syntax: предупреждения nginx -t."""
import os
import tempfile
import subprocess

from typer.testing import CliRunner

from commands.cli import app

runner = CliRunner()

_MINIMAL_CFG = """
events { worker_connections 1024; }
http { }
"""


def test_syntax_warns_exit_1_by_default(monkeypatch):
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".conf") as f:
        f.write(_MINIMAL_CFG)
        path = f.name

    def fake_run(cmd, **kwargs):
        class R:
            returncode = 0
            stdout = ""
            stderr = "nginx: [warn] directive is deprecated\n"

        return R()

    try:
        monkeypatch.setattr(subprocess, "run", fake_run)
        result = runner.invoke(app, ["syntax", "--config", path])
        assert result.exit_code == 1
        assert "предупреж" in result.output or "[warn]" in result.output
    finally:
        os.unlink(path)


def test_syntax_skip_warns_exits_0(monkeypatch):
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".conf") as f:
        f.write(_MINIMAL_CFG)
        path = f.name

    def fake_run(cmd, **kwargs):
        class R:
            returncode = 0
            stdout = ""
            stderr = "nginx: [warn] directive is deprecated\n"

        return R()

    try:
        monkeypatch.setattr(subprocess, "run", fake_run)
        result = runner.invoke(app, ["syntax", "--config", path, "--skip-warns"])
        assert result.exit_code == 0
        assert "skip" in result.output.lower() or "пропущ" in result.output.lower() or "коррект" in result.output.lower()
    finally:
        os.unlink(path)


def test_syntax_clean_ok(monkeypatch):
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".conf") as f:
        f.write(_MINIMAL_CFG)
        path = f.name

    def fake_run(cmd, **kwargs):
        class R:
            returncode = 0
            stdout = ""
            stderr = ""

        return R()

    try:
        monkeypatch.setattr(subprocess, "run", fake_run)
        result = runner.invoke(app, ["syntax", "--config", path])
        assert result.exit_code == 0
        assert "коррект" in result.output.lower() or "valid" in result.output.lower() or "Синтаксис" in result.output
    finally:
        os.unlink(path)
