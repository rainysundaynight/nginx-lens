# upstream_checker/checker.py

import socket
import time
import http.client
from typing import Dict, List


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


def resolve_address(address: str) -> List[str]:
    """
    Резолвит адрес upstream сервера в IP-адреса.
    
    Args:
        address: Адрес в формате "host:port" или "host:port параметры"
        
    Returns:
        Список IP-адресов в формате "ip:port" или пустой список, если резолвинг не удался
    """
    try:
        host_port = address.split()[0]
        
        if ":" not in host_port:
            return []
            
        parts = host_port.rsplit(":", 1)
        if len(parts) != 2:
            return []
        host, port = parts
        
        # Если это уже IPv4 адрес, возвращаем как есть
        try:
            socket.inet_aton(host)
            return [host_port]
        except socket.error:
            pass
        
        # Проверяем IPv6 (в квадратных скобках)
        if host.startswith("[") and host.endswith("]"):
            ipv6_host = host[1:-1]
            try:
                socket.inet_pton(socket.AF_INET6, ipv6_host)
                return [host_port]
            except (socket.error, OSError):
                pass
        
        # Пытаемся резолвить DNS имя - получаем все IP-адреса
        try:
            # gethostbyname_ex возвращает (hostname, aliaslist, ipaddrlist)
            _, _, ipaddrlist = socket.gethostbyname_ex(host)
            # Фильтруем только IPv4 адреса (IPv6 обрабатываются отдельно)
            resolved_ips = []
            for ip in ipaddrlist:
                try:
                    # Проверяем, что это IPv4
                    socket.inet_aton(ip)
                    resolved_ips.append(f"{ip}:{port}")
                except socket.error:
                    pass
            return resolved_ips if resolved_ips else []
        except (socket.gaierror, OSError):
            return []
    except (ValueError, IndexError, AttributeError):
        return []


def resolve_upstreams(
    upstreams: Dict[str, List[str]]
) -> Dict[str, List[dict]]:
    """
    Резолвит DNS имена upstream-серверов в IP-адреса.
    
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
    results = {}
    for name, servers in upstreams.items():
        results[name] = []
        for srv in servers:
            resolved = resolve_address(srv)
            results[name].append({
                "address": srv,
                "resolved": resolved
            })
    return results


def check_upstreams(
    upstreams: Dict[str, List[str]],
    timeout: float = 2.0,
    retries: int = 1,
    mode: str = "tcp"
) -> Dict[str, List[dict]]:
    """
    Проверяет доступность upstream-серверов.
    mode: "tcp" (по умолчанию) или "http"
    
    Возвращает:
    {
        "backend": [
            {"address": "127.0.0.1:8080", "healthy": True},
            ...
        ]
    }
    """
    results = {}
    for name, servers in upstreams.items():
        results[name] = []
        for srv in servers:
            if mode.lower() == "http":
                healthy = check_http(srv, timeout, retries)
            else:
                healthy = check_tcp(srv, timeout, retries)
            results[name].append({"address": srv, "healthy": healthy})
    return results
