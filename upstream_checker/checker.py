# upstream_checker/checker.py

import socket
import time
import http.client
from typing import Dict, List, Tuple, Optional
from concurrent.futures import ThreadPoolExecutor, as_completed
from utils.progress import ProgressManager
try:
    import dns.resolver
    import dns.exception
    DNS_AVAILABLE = True
except ImportError:
    DNS_AVAILABLE = False

from upstream_checker.dns_cache import get_cache, is_cache_enabled, disable_cache, enable_cache


def check_tcp(address: str, timeout: float, retries: int) -> bool:
    """
    Проверка доступности сервера по TCP.
    Ignores extra upstream options like 'max_fails' or 'fail_timeout'.
    """
    # Берем только host:port, игнорируем параметры
    host_port = address.split()[0]
    host, port = host_port.split(":")
    port = int(port)
    
    for _ in range(retries):
        try:
            with socket.create_connection((host, port), timeout=timeout):
                return True
        except (socket.timeout, ConnectionRefusedError, OSError):
            time.sleep(0.2)
    return False


def check_http(address: str, timeout: float, retries: int) -> bool:
    """
    Проверка доступности сервера по HTTP (GET /).
    Ignores extra upstream options like 'max_fails' or 'fail_timeout'.
    """
    host_port = address.split()[0]
    host, port = host_port.split(":")
    port = int(port)
    
    for _ in range(retries):
        try:
            conn = http.client.HTTPConnection(host, port, timeout=timeout)
            conn.request("GET", "/")
            resp = conn.getresponse()
            healthy = resp.status < 500
            conn.close()
            if healthy:
                return True
        except Exception:
            time.sleep(0.2)
            continue
    return False


def resolve_address(address: str, use_cache: bool = True, cache_ttl: int = 300, cache_dir: Optional[str] = None) -> List[str]:
    """
    Резолвит адрес upstream сервера в IP-адреса с информацией о CNAME.
    
    Args:
        address: Адрес в формате "host:port" или "host:port параметры"
        use_cache: Использовать ли кэш (по умолчанию True)
        cache_ttl: Время жизни кэша в секундах (по умолчанию 300)
        cache_dir: Директория для кэша (опционально)
        
    Returns:
        Список строк в формате:
        - "ip:port" - если резолвинг успешен без CNAME
        - "ip:port (via cname.example.com)" - если есть CNAME и все ок
        - "invalid resolve (via cname.example.com -> TXT)" - если CNAME ведет на невалидную запись
        Пустой список, если резолвинг не удался
    """
    try:
        host_port = address.split()[0]
        
        if ":" not in host_port:
            return []
            
        parts = host_port.rsplit(":", 1)
        if len(parts) != 2:
            return []
        host, port = parts
        
        # Проверка на IP адрес (не кэшируем IP адреса)
        try:
            socket.inet_aton(host)
            return [host_port]
        except socket.error:
            pass
        
        if host.startswith("[") and host.endswith("]"):
            ipv6_host = host[1:-1]
            try:
                socket.inet_pton(socket.AF_INET6, ipv6_host)
                return [host_port]
            except (socket.error, OSError):
                pass
        
        # Проверяем кэш перед резолвингом
        if use_cache and is_cache_enabled():
            cache = get_cache(ttl=cache_ttl, cache_dir=cache_dir)
            cached_result = cache.get(host, port)
            if cached_result is not None:
                return cached_result
        
        # Выполняем резолвинг
        if DNS_AVAILABLE:
            result = _resolve_with_dns(host, port)
        else:
            result = _resolve_with_socket(host, port)
        
        # Сохраняем в кэш
        if use_cache and is_cache_enabled():
            cache = get_cache(ttl=cache_ttl, cache_dir=cache_dir)
            cache.set(host, port, result)
        
        return result
    except (ValueError, IndexError, AttributeError):
        return []


def _resolve_with_dns(host: str, port: str) -> List[str]:
    """Резолвит DNS с использованием dnspython для получения информации о CNAME."""
    try:
        cname_info = None
        invalid_type = None
        
        try:
            cname_answer = dns.resolver.resolve(host, 'CNAME', raise_on_no_answer=False)
            if cname_answer:
                cname_target = str(cname_answer[0].target).rstrip('.')
                cname_info = cname_target
                
                try:
                    a_answer = dns.resolver.resolve(cname_target, 'A', raise_on_no_answer=False)
                    if a_answer:
                        resolved_ips = []
                        for rdata in a_answer:
                            ip = str(rdata.address)
                            resolved_ips.append(f"{ip}:{port} (via {cname_info})")
                        return resolved_ips
                    else:
                        try:
                            txt_answer = dns.resolver.resolve(cname_target, 'TXT', raise_on_no_answer=False)
                            if txt_answer:
                                invalid_type = 'TXT'
                        except:
                            pass
                        if not invalid_type:
                            try:
                                mx_answer = dns.resolver.resolve(cname_target, 'MX', raise_on_no_answer=False)
                                if mx_answer:
                                    invalid_type = 'MX'
                            except:
                                pass
                        if not invalid_type:
                            try:
                                ns_answer = dns.resolver.resolve(cname_target, 'NS', raise_on_no_answer=False)
                                if ns_answer:
                                    invalid_type = 'NS'
                            except:
                                pass
                        if invalid_type:
                            return [f"invalid resolve (via {cname_info} -> {invalid_type})"]
                        else:
                            return [f"invalid resolve (via {cname_info})"]
                except Exception:
                    return [f"invalid resolve (via {cname_info})"]
        except Exception:
            pass
        
        try:
            a_answer = dns.resolver.resolve(host, 'A', raise_on_no_answer=False)
            if a_answer:
                resolved_ips = []
                for rdata in a_answer:
                    ip = str(rdata.address)
                    resolved_ips.append(f"{ip}:{port}")
                return resolved_ips if resolved_ips else []
        except Exception:
            pass
        
        return []
    except Exception:
        return []


def _resolve_with_socket(host: str, port: str) -> List[str]:
    """Fallback резолвинг через socket (без информации о CNAME)."""
    try:
        _, _, ipaddrlist = socket.gethostbyname_ex(host)
        resolved_ips = []
        for ip in ipaddrlist:
            try:
                socket.inet_aton(ip)
                resolved_ips.append(f"{ip}:{port}")
            except socket.error:
                pass
        return resolved_ips if resolved_ips else []
    except (socket.gaierror, OSError):
        return []


def resolve_upstreams(
    upstreams: Dict[str, List[str]],
    max_workers: int = 10,
    use_cache: bool = True,
    cache_ttl: int = 300,
    cache_dir: Optional[str] = None,
    progress_manager: Optional[ProgressManager] = None
) -> Dict[str, List[dict]]:
    """
    Резолвит DNS имена upstream-серверов в IP-адреса.
    
    Args:
        upstreams: Словарь upstream серверов
        max_workers: Максимальное количество потоков для параллельной обработки
        use_cache: Использовать ли кэш (по умолчанию True)
        cache_ttl: Время жизни кэша в секундах (по умолчанию 300)
        cache_dir: Директория для кэша (опционально)
    
    Возвращает:
    {
        "backend": [
            {"address": "example.com:8080", "resolved": ["192.168.1.1:8080", "192.168.1.2:8080"]},
            {"address": "127.0.0.1:8080", "resolved": ["127.0.0.1:8080"]},
            {"address": "badhost:80", "resolved": []},
            ...
        ]
    }
    """
    # Собираем все задачи для параллельной обработки
    tasks = []
    task_to_key = {}
    
    for name, servers in upstreams.items():
        for idx, srv in enumerate(servers):
            key = (name, idx, srv)
            tasks.append((key, srv))
            task_to_key[key] = (name, idx)
    
    results = {}
    for name in upstreams.keys():
        results[name] = [None] * len(upstreams[name])
    
    # Если нет задач, возвращаем пустой результат
    if not tasks:
        return results
    
    # Параллельная обработка резолвинга
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        future_to_key = {
            executor.submit(resolve_address, srv, use_cache, cache_ttl, cache_dir): key 
            for key, srv in tasks
        }
        
        for future in as_completed(future_to_key):
            key = future_to_key[future]
            name, idx = task_to_key[key]
            try:
                resolved = future.result()
                results[name][idx] = {
                    "address": key[2],
                    "resolved": resolved
                }
            except Exception:
                results[name][idx] = {
                    "address": key[2],
                    "resolved": []
                }
    
    return results


def _check_single_upstream(srv: str, timeout: float, retries: int, mode: str) -> Tuple[str, bool]:
    """Вспомогательная функция для проверки одного upstream сервера."""
    if mode.lower() == "http":
        healthy = check_http(srv, timeout, retries)
    else:
        healthy = check_tcp(srv, timeout, retries)
    return (srv, healthy)


def check_upstreams(
    upstreams: Dict[str, List[str]],
    timeout: float = 2.0,
    retries: int = 1,
    mode: str = "tcp",
    max_workers: int = 10,
    progress_manager: Optional[ProgressManager] = None
) -> Dict[str, List[dict]]:
    """
    Проверяет доступность upstream-серверов.
    
    Args:
        upstreams: Словарь upstream серверов
        timeout: Таймаут проверки (сек)
        retries: Количество попыток
        mode: "tcp" (по умолчанию) или "http"
        max_workers: Максимальное количество потоков для параллельной обработки
    
    Возвращает:
    {
        "backend": [
            {"address": "127.0.0.1:8080", "healthy": True},
            ...
        ]
    }
    """
    # Собираем все задачи для параллельной обработки
    tasks = []
    task_to_key = {}
    
    for name, servers in upstreams.items():
        for idx, srv in enumerate(servers):
            key = (name, idx, srv)
            tasks.append((key, srv))
            task_to_key[key] = (name, idx)
    
    results = {}
    for name in upstreams.keys():
        results[name] = [None] * len(upstreams[name])
    
    # Если нет задач, возвращаем пустой результат
    if not tasks:
        return results
    
    # Параллельная обработка проверок
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        future_to_key = {
            executor.submit(_check_single_upstream, srv, timeout, retries, mode): key 
            for key, srv in tasks
        }
        
        completed = 0
        total = len(tasks)
        
        for future in as_completed(future_to_key):
            key = future_to_key[future]
            name, idx = task_to_key[key]
            try:
                srv, healthy = future.result()
                results[name][idx] = {"address": srv, "healthy": healthy}
            except Exception:
                results[name][idx] = {"address": key[2], "healthy": False}
            
            completed += 1
            if progress_manager:
                progress_manager.update(completed, total=total, description=f"Проверка upstream серверов ({completed}/{total})")
    
    return results
