from typer.testing import CliRunner

from commands.cli import app

runner = CliRunner()


def test_version_flag_exits_zero_and_prints_name():
    r = runner.invoke(app, ["--version"])
    assert r.exit_code == 0
    out = (r.output or r.stdout or "").lower()
    assert "nginx-lens" in out
    assert "0.7.1" in out or "0.7" in out  # fallback или metadata


def test_version_short_v():
    r = runner.invoke(app, ["-V"])
    assert r.exit_code == 0
    assert "nginx-lens" in (r.output or r.stdout or "")
