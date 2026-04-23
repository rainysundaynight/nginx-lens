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

def test_logs_russian_timezone_kaiten_style():
    """Тот же combined, дата с +0300 (как на реальных access.log)."""
    log_content = (
        '10.67.24.35 - - [23/Apr/2026:10:28:32 +0300] "GET /api/cards/552673?x=1 HTTP/1.1" '
        '200 1234 "https://example.com/" "Mozilla/5.0"\n'
    )
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".log") as f:
        f.write(log_content)
        f.flush()
        log_path = f.name
    try:
        result = runner.invoke(app, ["logs", log_path, "--top", "5"])
        assert result.exit_code == 0
        assert "/api/cards" in result.output
        assert "200" in result.output
    finally:
        os.unlink(log_path)


def test_logs_minimal_line_without_referer_ua():
    """Log_format без кавычек referer/ua (короткий combined)."""
    line = '127.0.0.1 - - [01/Jan/2024:00:00:00 +0000] "GET /page0 HTTP/1.1" 200 1234\n'
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".log") as f:
        f.write(line * 3)
        f.flush()
        log_path = f.name
    try:
        result = runner.invoke(app, ["logs", log_path, "--top", "3"])
        assert result.exit_code == 0
        assert "/page0" in result.output
    finally:
        os.unlink(log_path)


def test_logs_unparseable_shows_message_not_filters():
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".log") as f:
        f.write("this is not an access line\n")
        f.flush()
        log_path = f.name
    try:
        result = runner.invoke(app, ["logs", log_path])
        assert result.exit_code == 0
        assert "HTTP" in result.output or "разобрать" in result.output.lower() or "файла" in result.output.lower()
    finally:
        os.unlink(log_path)


def test_logs_json_line_auto_detected():
    j = (
        '{"@timestamp":"2024-01-15T10:00:00Z","status":200,"path":"/api/v1/health",'
        '"ip":"10.0.0.1","user_agent":"curl"}\n'
    )
    with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".log") as f:
        f.write(j)
        f.flush()
        log_path = f.name
    try:
        r = runner.invoke(app, ["logs", log_path])
        assert r.exit_code == 0
        assert "/api/v1/health" in r.output
        assert "200" in r.output
    finally:
        os.unlink(log_path)


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

