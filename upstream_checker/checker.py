# upstream_checker/checker.py

import socket
import time
import http.client
from typing import Dict, List
try:
    import dns.resolver
    import dns.exception
    DNS_AVAILABLE = True
except ImportError:
    DNS_AVAILABLE = False


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
    Резолвит адрес upstream сервера в IP-адреса с информацией о CNAME.
    
    Args:
        address: Адрес в формате "host:port" или "host:port параметры"
        
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
        
        if DNS_AVAILABLE:
            return _resolve_with_dns(host, port)
        else:
            return _resolve_with_socket(host, port)
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
