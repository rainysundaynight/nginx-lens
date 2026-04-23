import sys

import pytest

from commands.cli import main


def test_version_prints_and_exits_0(capsys):
    old = sys.argv
    try:
        sys.argv = ["nginx-lens", "--version"]
        with pytest.raises(SystemExit) as e:
            main()
        assert e.value.code == 0
    finally:
        sys.argv = old
    out = capsys.readouterr().out
    assert "nginx-lens" in out
    assert "0.8" in out


def test_version_short_v(capsys):
    old = sys.argv
    try:
        sys.argv = ["nginx-lens", "-V"]
        with pytest.raises(SystemExit) as e:
            main()
        assert e.value.code == 0
    finally:
        sys.argv = old
    assert "nginx-lens" in capsys.readouterr().out
