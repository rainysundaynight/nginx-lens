"""
Модуль для работы с HTTP API ngx_dynamic_upstream.

Позволяет получать список серверов из динамических upstream блоков
через HTTP API модуля.
"""
import re
import urllib.request
import urllib.error
import urllib.parse
from typing import Dict, List, Optional, Tuple, Any
import socket


def get_dynamic_upstream_servers(
    upstream_name: str,
    api_url: str,
    timeout: float = 2.0
) -> Tuple[List[str], Optional[str]]:
    """
    Получает список серверов из динамического upstream через HTTP API модуля.
    
    Args:
        upstream_name: Имя upstream блока
        api_url: URL endpoint модуля (например, "http://127.0.0.1:6000/dynamic")
        timeout: Таймаут HTTP запроса в секундах
        
    Returns:
        Кортеж (список серверов, сообщение об ошибке или None)
        
    Пример запроса:
        GET http://127.0.0.1:6000/dynamic?upstream=backends2
        
    Пример ответа:
        server 127.0.0.1:6001;
        server 127.0.0.1:6002;
        server 127.0.0.1:6003;
    """
    try:
        # Формируем URL запроса
        parsed_url = urllib.parse.urlparse(api_url)
        query_params = urllib.parse.parse_qs(parsed_url.query)
        query_params['upstream'] = [upstream_name]
        
        # Собираем URL обратно
        new_query = urllib.parse.urlencode(query_params, doseq=True)
        request_url = urllib.parse.urlunparse((
            parsed_url.scheme,
            parsed_url.netloc,
            parsed_url.path,
            parsed_url.params,
            new_query,
            parsed_url.fragment
        ))
        
        # Выполняем HTTP запрос
        req = urllib.request.Request(request_url)
        req.add_header('User-Agent', 'nginx-lens/1.0')
        
        with urllib.request.urlopen(req, timeout=timeout) as response:
            response_text = response.read().decode('utf-8')
            
            # Парсим ответ модуля
            # Формат: "server addr:port; server addr:port; ..."
            servers = []
            for line in response_text.split('\n'):
                line = line.strip()
                if not line:
                    continue
                    
                # Ищем строки вида "server addr:port;" или "server addr:port параметры;"
                match = re.match(r'server\s+([^;]+);', line)
                if match:
                    server_spec = match.group(1).strip()
                    # Извлекаем адрес (до первого пробела или до конца)
                    # Может быть "127.0.0.1:6001" или "mail.ru" или "127.0.0.1:6001 backup"
                    address = server_spec.split()[0].strip()
                    if address:
                        servers.append(address)
            
            return servers, None
            
    except urllib.error.HTTPError as e:
        if e.code == 403:
            return [], f"Доступ запрещен (403). Проверьте настройки allow/deny в location /dynamic"
        elif e.code == 404:
            return [], f"Endpoint не найден (404). Проверьте путь к API: {api_url}"
        else:
            return [], f"HTTP ошибка {e.code}: {e.reason}"
    except urllib.error.URLError as e:
        if isinstance(e.reason, socket.timeout):
            return [], f"Таймаут подключения к {api_url}"
        elif isinstance(e.reason, ConnectionRefusedError):
            return [], f"Подключение отклонено. Проверьте, что nginx запущен и доступен по адресу {api_url}"
        else:
            return [], f"Ошибка подключения: {e.reason}"
    except socket.timeout:
        return [], f"Таймаут подключения к {api_url}"
    except Exception as e:
        return [], f"Неожиданная ошибка: {str(e)}"


def get_dynamic_upstream_servers_verbose(
    upstream_name: str,
    api_url: str,
    timeout: float = 2.0
) -> Tuple[List[Dict[str, Any]], Optional[str]]:
    """
    Получает детальную информацию о серверах через verbose API.
    
    Args:
        upstream_name: Имя upstream блока
        api_url: URL endpoint модуля
        timeout: Таймаут HTTP запроса в секундах
        
    Returns:
        Кортеж (список словарей с информацией о серверах, сообщение об ошибке или None)
        
    Пример запроса:
        GET http://127.0.0.1:6000/dynamic?upstream=backends&verbose=
        
    Пример ответа:
        server 127.0.0.1:6001 weight=1 max_fails=1 fail_timeout=10 max_conns=0 conns=0;
        server 127.0.0.1:6002 weight=1 max_fails=1 fail_timeout=10 max_conns=0 conns=0;
    """
    try:
        # Формируем URL запроса с verbose
        parsed_url = urllib.parse.urlparse(api_url)
        query_params = urllib.parse.parse_qs(parsed_url.query)
        query_params['upstream'] = [upstream_name]
        query_params['verbose'] = ['']
        
        # Собираем URL обратно
        new_query = urllib.parse.urlencode(query_params, doseq=True)
        request_url = urllib.parse.urlunparse((
            parsed_url.scheme,
            parsed_url.netloc,
            parsed_url.path,
            parsed_url.params,
            new_query,
            parsed_url.fragment
        ))
        
        # Выполняем HTTP запрос
        req = urllib.request.Request(request_url)
        req.add_header('User-Agent', 'nginx-lens/1.0')
        
        with urllib.request.urlopen(req, timeout=timeout) as response:
            response_text = response.read().decode('utf-8')
            
            # Парсим ответ модуля
            servers_info = []
            for line in response_text.split('\n'):
                line = line.strip()
                if not line:
                    continue
                    
                # Формат: "server addr:port weight=1 max_fails=1 fail_timeout=10 max_conns=0 conns=0;"
                match = re.match(r'server\s+([^;]+);', line)
                if match:
                    server_spec = match.group(1).strip()
                    parts = server_spec.split()
                    if not parts:
                        continue
                        
                    address = parts[0]
                    
                    # Парсим параметры
                    params = {}
                    for part in parts[1:]:
                        if '=' in part:
                            key, value = part.split('=', 1)
                            try:
                                # Пробуем преобразовать в число
                                if '.' in value:
                                    params[key] = float(value)
                                else:
                                    params[key] = int(value)
                            except ValueError:
                                params[key] = value
                        elif part in ['backup', 'down']:
                            params[part] = True
                    
                    servers_info.append({
                        'address': address,
                        'params': params
                    })
            
            return servers_info, None
            
    except Exception as e:
        return [], f"Ошибка при получении verbose информации: {str(e)}"


def enrich_upstreams_with_dynamic(
    upstreams: Dict[str, List[str]],
    api_url: Optional[str] = None,
    timeout: float = 2.0,
    enabled: bool = True
) -> Dict[str, List[str]]:
    """
    Обогащает словарь upstream серверами из динамических upstream через API.
    
    Args:
        upstreams: Словарь {upstream_name: [servers]} из парсера
        api_url: URL endpoint модуля (если None, используется из конфига)
        timeout: Таймаут HTTP запросов
        enabled: Включена ли интеграция с dynamic upstream
        
    Returns:
        Обогащенный словарь upstream с добавленными динамическими серверами
    """
    if not enabled or not api_url:
        return upstreams
    
    enriched = upstreams.copy()
    
    # Для каждого upstream, который не имеет серверов в конфиге,
    # пытаемся получить их через API
    for upstream_name, servers in upstreams.items():
        if not servers:  # Пустой список - возможно, динамический upstream
            dynamic_servers, error = get_dynamic_upstream_servers(
                upstream_name,
                api_url,
                timeout
            )
            if error:
                # Сохраняем ошибку для отображения пользователю
                # Пока просто пропускаем
                continue
            if dynamic_servers:
                enriched[upstream_name] = dynamic_servers
    
    return enriched

