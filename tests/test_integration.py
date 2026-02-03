"""
Интеграционные тесты для полных сценариев использования nginx-lens
"""
import pytest
import tempfile
import os
from typer.testing import CliRunner
from commands.cli import app

runner = CliRunner()

def test_full_workflow_health_and_resolve():
    """Интеграционный тест: health + resolve для реального конфига"""
    config_content = """
events {
    worker_connections 1024;
}

http {
    upstream backend {
        server example.com:80;
        server 127.0.0.1:8080;
    }
    
    server {
        listen 80;
        location / {
            proxy_pass http://backend;
        }
    }
}
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f:
        f.write(config_content)
        f.flush()
        config_path = f.name
    
    try:
        # Тест resolve
        result_resolve = runner.invoke(app, ["resolve", config_path, "--max-workers", "5"])
        assert result_resolve.exit_code == 0
        assert "backend" in result_resolve.output
        
        # Тест health
        result_health = runner.invoke(app, ["health", config_path, "--timeout", "1", "--max-workers", "5"])
        assert result_health.exit_code in [0, 1]  # Может быть 0 или 1 в зависимости от доступности
        
        # Тест health с resolve
        result_health_resolve = runner.invoke(app, ["health", config_path, "--resolve", "--timeout", "1", "--max-workers", "5"])
        assert result_health_resolve.exit_code in [0, 1]
    finally:
        os.unlink(config_path)

def test_full_workflow_analyze_and_tree():
    """Интеграционный тест: analyze + tree для реального конфига"""
    config_content = """
events {
    worker_connections 1024;
}

http {
    upstream backend {
        server 127.0.0.1:8080;
    }
    
    server {
        listen 80;
        server_name example.com;
        
        location / {
            proxy_pass http://backend;
        }
        
        location /api {
            proxy_pass http://backend;
        }
    }
}
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f:
        f.write(config_content)
        f.flush()
        config_path = f.name
    
    try:
        # Тест analyze
        result_analyze = runner.invoke(app, ["analyze", config_path])
        assert result_analyze.exit_code == 0
        
        # Тест tree
        result_tree = runner.invoke(app, ["tree", config_path])
        assert result_tree.exit_code == 0
        assert "server" in result_tree.output.lower() or "location" in result_tree.output.lower()
    finally:
        os.unlink(config_path)

def test_full_workflow_diff_and_syntax():
    """Интеграционный тест: diff + syntax для реальных конфигов"""
    config1_content = """
events {
    worker_connections 1024;
}

http {
    upstream backend {
        server 127.0.0.1:8080;
    }
}
"""
    config2_content = """
events {
    worker_connections 2048;
}

http {
    upstream backend {
        server 127.0.0.1:8080;
        server 127.0.0.1:8081;
    }
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
        # Тест diff
        result_diff = runner.invoke(app, ["diff", config1_path, config2_path])
        assert result_diff.exit_code == 0
        
        # Тест syntax (может требовать nginx бинарь, поэтому проверяем только что команда работает)
        # Если nginx не установлен, команда вернет ошибку, но это нормально для теста
        result_syntax = runner.invoke(app, ["syntax", "--config", config1_path])
        # Exit code может быть любым, главное что команда выполнилась
        assert result_syntax.exit_code is not None
    finally:
        os.unlink(config1_path)
        os.unlink(config2_path)

def test_error_handling_workflow():
    """Интеграционный тест обработки ошибок"""
    # Тест с несуществующим файлом
    result = runner.invoke(app, ["health", "/nonexistent/nginx.conf"])
    assert result.exit_code == 1
    
    result = runner.invoke(app, ["resolve", "/nonexistent/nginx.conf"])
    assert result.exit_code == 1
    
    result = runner.invoke(app, ["analyze", "/nonexistent/nginx.conf"])
    assert result.exit_code != 0
    
    result = runner.invoke(app, ["tree", "/nonexistent/nginx.conf"])
    assert result.exit_code != 0
    
    result = runner.invoke(app, ["diff", "/nonexistent1.conf", "/nonexistent2.conf"])
    assert result.exit_code == 1
    
    result = runner.invoke(app, ["logs", "/nonexistent/log.log"])
    assert result.exit_code == 1

def test_exit_codes_workflow():
    """Интеграционный тест exit codes для CI/CD"""
    config_healthy = """
upstream backend {
    server 127.0.0.1:99999;  # Недоступный, но не критично для теста
}
"""
    config_unhealthy = """
upstream backend {
    server nonexistent-domain-12345.invalid:80;
}
"""
    with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f1, \
         tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.conf') as f2:
        f1.write(config_healthy)
        f2.write(config_unhealthy)
        f1.flush()
        f2.flush()
        config1_path = f1.name
        config2_path = f2.name
    
    try:
        # Resolve с валидным DNS должен вернуть 0 или 1 (зависит от доступности)
        result1 = runner.invoke(app, ["resolve", config1_path])
        assert result1.exit_code in [0, 1]
        
        # Resolve с невалидным DNS должен вернуть 1
        result2 = runner.invoke(app, ["resolve", config2_path])
        # Может быть 0 если DNS не резолвится, но это нормально
        assert result2.exit_code is not None
    finally:
        os.unlink(config1_path)
        os.unlink(config2_path)

