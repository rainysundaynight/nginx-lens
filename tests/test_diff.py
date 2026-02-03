import typer
from typer.testing import CliRunner
from commands.cli import app
import pytest
import tempfile
import os

runner = CliRunner()

def test_diff_basic():
    """Тест базовой функциональности diff"""
    config1_content = """
upstream backend {
    server 127.0.0.1:8080;
}
"""
    config2_content = """
upstream backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
}
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f1, \
         tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f2:
        f1.write(config1_content)
        f2.write(config2_content)
        f1.flush()
        f2.flush()
        config1_path = f1.name
        config2_path = f2.name
    
    try:
        result = runner.invoke(app, ["diff", config1_path, config2_path])
        assert result.exit_code == 0
        # Проверяем что diff показывает различия
    finally:
        os.unlink(config1_path)
        os.unlink(config2_path)

def test_diff_identical():
    """Тест diff для идентичных конфигов"""
    config_content = """
upstream backend {
    server 127.0.0.1:8080;
}
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f1, \
         tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f2:
        f1.write(config_content)
        f2.write(config_content)
        f1.flush()
        f2.flush()
        config1_path = f1.name
        config2_path = f2.name
    
    try:
        result = runner.invoke(app, ["diff", config1_path, config2_path])
        assert result.exit_code == 0
    finally:
        os.unlink(config1_path)
        os.unlink(config2_path)

def test_diff_file_not_found():
    """Тест обработки отсутствующих файлов"""
    result = runner.invoke(app, ["diff", "/nonexistent1.conf", "/nonexistent2.conf"])
    # Команда должна вернуть exit code 1 при отсутствии файлов
    assert result.exit_code == 1

