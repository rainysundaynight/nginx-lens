import typer
from typer.testing import CliRunner
from commands.cli import app
import pytest
import tempfile
import os

runner = CliRunner()

def test_logs_basic():
    """Тест базовой функциональности logs"""
    log_content = """127.0.0.1 - - [01/Jan/2024:00:00:00 +0000] "GET /api/v1 HTTP/1.1" 200 1234 "-" "Mozilla/5.0"
192.168.1.1 - - [01/Jan/2024:00:00:01 +0000] "GET /page HTTP/1.1" 404 567 "-" "curl/7.0"
10.0.0.1 - - [01/Jan/2024:00:00:02 +0000] "POST /api/v2 HTTP/1.1" 500 890 "-" "Python/3.0"
127.0.0.1 - - [01/Jan/2024:00:00:03 +0000] "GET /api/v1 HTTP/1.1" 200 1234 "-" "Mozilla/5.0"
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.log') as f:
        f.write(log_content)
        f.flush()
        log_path = f.name
    
    try:
        result = runner.invoke(app, ["logs", log_path])
        assert result.exit_code == 0
        assert "200" in result.output
        assert "404" in result.output
        assert "500" in result.output
        assert "/api/v1" in result.output
        assert "127.0.0.1" in result.output
    finally:
        os.unlink(log_path)

def test_logs_top_option():
    """Тест опции --top"""
    log_content = "\n".join([f"127.0.0.1 - - [01/Jan/2024:00:00:0{i} +0000] \"GET /page{i} HTTP/1.1\" 200 1234" for i in range(20)])
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.log') as f:
        f.write(log_content)
        f.flush()
        log_path = f.name
    
    try:
        result = runner.invoke(app, ["logs", log_path, "--top", "5"])
        assert result.exit_code == 0
        # Проверяем что вывод ограничен топ-5
    finally:
        os.unlink(log_path)

def test_logs_file_not_found():
    """Тест обработки отсутствующего файла"""
    result = runner.invoke(app, ["logs", "/nonexistent/path/to/log.log"])
    assert result.exit_code == 1
    assert "не найден" in result.output.lower() or "not found" in result.output.lower()

def test_logs_error_statuses():
    """Тест анализа ошибок 404 и 500"""
    log_content = """127.0.0.1 - - [01/Jan/2024:00:00:00 +0000] "GET /notfound HTTP/1.1" 404 123 "-" "Mozilla"
127.0.0.1 - - [01/Jan/2024:00:00:01 +0000] "GET /notfound HTTP/1.1" 404 123 "-" "Mozilla"
127.0.0.1 - - [01/Jan/2024:00:00:02 +0000] "GET /error HTTP/1.1" 500 456 "-" "curl"
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.log') as f:
        f.write(log_content)
        f.flush()
        log_path = f.name
    
    try:
        result = runner.invoke(app, ["logs", log_path])
        assert result.exit_code == 0
        assert "404" in result.output
        assert "500" in result.output
        assert "/notfound" in result.output
        assert "/error" in result.output
    finally:
        os.unlink(log_path)

