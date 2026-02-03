import pytest
import time
from upstream_checker.checker import check_upstreams, resolve_upstreams

def test_parallel_processing_performance():
    """Тест производительности параллельной обработки"""
    # Создаем большое количество upstream для теста
    upstreams = {}
    for i in range(30):
        upstreams[f"backend{i}"] = [
            "127.0.0.1:99999",  # Недоступный порт для быстрого таймаута
            "example.com:80",
            "google.com:80"
        ]
    
    # Тест с max_workers=1 (последовательно)
    start_time = time.time()
    results_sequential = check_upstreams(upstreams, timeout=0.5, retries=1, max_workers=1)
    sequential_time = time.time() - start_time
    
    # Тест с max_workers=10 (параллельно)
    start_time = time.time()
    results_parallel = check_upstreams(upstreams, timeout=0.5, retries=1, max_workers=10)
    parallel_time = time.time() - start_time
    
    # Параллельная обработка должна быть быстрее
    assert parallel_time < sequential_time, f"Parallel ({parallel_time:.2f}s) should be faster than sequential ({sequential_time:.2f}s)"
    
    # Результаты должны быть одинаковыми
    assert len(results_sequential) == len(results_parallel)
    for name in upstreams.keys():
        assert len(results_sequential[name]) == len(results_parallel[name])

def test_resolve_parallel_performance(monkeypatch):
    """Тест производительности параллельного резолвинга"""
    import socket
    
    # Мокаем socket.gethostbyname_ex для быстрого теста
    original_gethostbyname_ex = socket.gethostbyname_ex
    
    def mock_gethostbyname_ex(host):
        time.sleep(0.01)  # Имитируем задержку DNS
        return (host, [], ["127.0.0.1"])
    
    monkeypatch.setattr(socket, "gethostbyname_ex", mock_gethostbyname_ex)
    
    upstreams = {}
    for i in range(20):
        upstreams[f"backend{i}"] = [f"example{i}.com:80"]
    
    # Тест с max_workers=1
    start_time = time.time()
    results_sequential = resolve_upstreams(upstreams, max_workers=1)
    sequential_time = time.time() - start_time
    
    # Тест с max_workers=10
    start_time = time.time()
    results_parallel = resolve_upstreams(upstreams, max_workers=10)
    parallel_time = time.time() - start_time
    
    # Параллельная обработка должна быть значительно быстрее
    assert parallel_time < sequential_time * 0.5, f"Parallel ({parallel_time:.2f}s) should be much faster than sequential ({sequential_time:.2f}s)"
    
    # Результаты должны быть одинаковыми
    assert len(results_sequential) == len(results_parallel)

def test_max_workers_scaling():
    """Тест масштабирования с разным количеством потоков"""
    upstreams = {"backend": ["127.0.0.1:99999"] * 10}
    
    times = {}
    for workers in [1, 5, 10, 20]:
        start_time = time.time()
        check_upstreams(upstreams, timeout=0.1, retries=1, max_workers=workers)
        times[workers] = time.time() - start_time
    
    # Больше потоков должно быть быстрее (до определенного предела)
    assert times[10] <= times[1], "10 workers should be faster than 1 worker"
    assert times[20] <= times[1], "20 workers should be faster than 1 worker"

