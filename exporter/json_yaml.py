"""
Модуль для экспорта результатов команд nginx-lens в JSON и YAML форматы.
"""
import json
import sys
from typing import Any, Dict, List, Optional
from datetime import datetime

try:
    import yaml
    YAML_AVAILABLE = True
except ImportError:
    YAML_AVAILABLE = False


def export_json(data: Any, pretty: bool = True) -> str:
    """
    Экспортирует данные в JSON формат.
    
    Args:
        data: Данные для экспорта
        pretty: Форматировать с отступами
        
    Returns:
        JSON строка
    """
    if pretty:
        return json.dumps(data, ensure_ascii=False, indent=2, default=str)
    else:
        return json.dumps(data, ensure_ascii=False, default=str)


def export_yaml(data: Any) -> str:
    """
    Экспортирует данные в YAML формат.
    
    Args:
        data: Данные для экспорта
        
    Returns:
        YAML строка
        
    Raises:
        ImportError: Если PyYAML не установлен
    """
    if not YAML_AVAILABLE:
        raise ImportError("PyYAML не установлен. Установите его: pip install pyyaml")
    
    return yaml.dump(data, allow_unicode=True, default_flow_style=False, sort_keys=False)


def print_export(data: Any, format_type: str, file=None):
    """
    Выводит данные в указанном формате.
    
    Args:
        data: Данные для экспорта
        format_type: 'json' или 'yaml'
        file: Файл для вывода (по умолчанию stdout)
    """
    if file is None:
        file = sys.stdout
    
    if format_type == 'json':
        output = export_json(data)
    elif format_type == 'yaml':
        output = export_yaml(data)
    else:
        raise ValueError(f"Неподдерживаемый формат: {format_type}")
    
    print(output, file=file)


def format_health_results(results: Dict[str, List[Dict]], resolved_info: Optional[Dict] = None) -> Dict[str, Any]:
    """
    Форматирует результаты команды health для экспорта.
    
    Args:
        results: Результаты check_upstreams
        resolved_info: Результаты resolve_upstreams (опционально)
        
    Returns:
        Словарь с данными для экспорта
    """
    data = {
        "timestamp": datetime.now().isoformat(),
        "upstreams": []
    }
    
    for name, servers in results.items():
        upstream_data = {
            "name": name,
            "servers": []
        }
        
        for srv in servers:
            server_data = {
                "address": srv["address"],
                "healthy": srv["healthy"],
                "status": "healthy" if srv["healthy"] else "unhealthy"
            }
            
            if resolved_info and name in resolved_info:
                for resolved_srv in resolved_info[name]:
                    if resolved_srv["address"] == srv["address"]:
                        server_data["resolved_ips"] = resolved_srv["resolved"]
                        # Проверяем наличие invalid resolve
                        server_data["has_invalid_resolve"] = any("invalid resolve" in r for r in resolved_srv["resolved"])
                        break
            
            upstream_data["servers"].append(server_data)
        
        data["upstreams"].append(upstream_data)
    
    # Подсчитываем статистику
    total_servers = sum(len(servers) for servers in results.values())
    healthy_count = sum(1 for servers in results.values() for srv in servers if srv["healthy"])
    unhealthy_count = total_servers - healthy_count
    
    data["summary"] = {
        "total_upstreams": len(results),
        "total_servers": total_servers,
        "healthy": healthy_count,
        "unhealthy": unhealthy_count
    }
    
    return data


def format_resolve_results(results: Dict[str, List[Dict]]) -> Dict[str, Any]:
    """
    Форматирует результаты команды resolve для экспорта.
    
    Args:
        results: Результаты resolve_upstreams
        
    Returns:
        Словарь с данными для экспорта
    """
    data = {
        "timestamp": datetime.now().isoformat(),
        "upstreams": []
    }
    
    failed_count = 0
    invalid_count = 0
    
    for name, servers in results.items():
        upstream_data = {
            "name": name,
            "servers": []
        }
        
        for srv in servers:
            server_data = {
                "address": srv["address"],
                "resolved_ips": srv["resolved"]
            }
            
            if not srv["resolved"]:
                server_data["status"] = "failed"
                failed_count += 1
            elif any("invalid resolve" in r for r in srv["resolved"]):
                server_data["status"] = "invalid"
                invalid_count += 1
            else:
                server_data["status"] = "success"
            
            upstream_data["servers"].append(server_data)
        
        data["upstreams"].append(upstream_data)
    
    data["summary"] = {
        "total_upstreams": len(results),
        "total_servers": sum(len(servers) for servers in results.values()),
        "successful": sum(len(servers) for servers in results.values()) - failed_count - invalid_count,
        "failed": failed_count,
        "invalid": invalid_count
    }
    
    return data


def format_analyze_results(
    conflicts: List[Dict],
    dups: List[Dict],
    empties: List[Dict],
    warnings: List[Dict],
    unused_vars: List[Dict],
    listen_conflicts: List[Dict],
    rewrite_issues: List[Dict],
    dead_locations: List[Dict],
    issue_meta: Dict[str, tuple]
) -> Dict[str, Any]:
    """
    Форматирует результаты команды analyze для экспорта.
    
    Args:
        conflicts: Конфликты location
        dups: Дублирующиеся директивы
        empties: Пустые блоки
        warnings: Предупреждения
        unused_vars: Неиспользуемые переменные
        listen_conflicts: Конфликты listen/server_name
        rewrite_issues: Проблемы с rewrite
        dead_locations: Мертвые location
        issue_meta: Метаданные о типах проблем
        
    Returns:
        Словарь с данными для экспорта
    """
    data = {
        "timestamp": datetime.now().isoformat(),
        "issues": []
    }
    
    # Добавляем все проблемы в единый список
    for c in conflicts:
        advice, severity = issue_meta.get('location_conflict', ("", "medium"))
        data["issues"].append({
            "type": "location_conflict",
            "severity": severity,
            "message": c.get('value', ''),
            "advice": advice
        })
    
    for d in dups:
        advice, severity = issue_meta.get('duplicate_directive', ("", "medium"))
        data["issues"].append({
            "type": "duplicate_directive",
            "severity": severity,
            "message": d.get('value', ''),
            "advice": advice
        })
    
    for e in empties:
        advice, severity = issue_meta.get('empty_block', ("", "low"))
        data["issues"].append({
            "type": "empty_block",
            "severity": severity,
            "message": f"{e.get('block', '')} блок пуст",
            "advice": advice
        })
    
    for w in warnings:
        issue_type = w.get('type', '')
        advice, severity = issue_meta.get(issue_type, ("", "medium"))
        data["issues"].append({
            "type": issue_type,
            "severity": severity,
            "message": w.get('value', ''),
            "advice": advice
        })
    
    for v in unused_vars:
        advice, severity = issue_meta.get('unused_variable', ("", "low"))
        data["issues"].append({
            "type": "unused_variable",
            "severity": severity,
            "message": v.get('name', ''),
            "advice": advice
        })
    
    for c in listen_conflicts:
        advice, severity = issue_meta.get('listen_servername_conflict', ("", "high"))
        data["issues"].append({
            "type": "listen_servername_conflict",
            "severity": severity,
            "message": f"server1: {c.get('server1', {}).get('arg','')} server2: {c.get('server2', {}).get('arg','')}",
            "advice": advice
        })
    
    for r in rewrite_issues:
        advice, severity = issue_meta.get(r.get('type', ''), ("", "medium"))
        data["issues"].append({
            "type": r.get('type', ''),
            "severity": severity,
            "message": r.get('value', ''),
            "advice": advice
        })
    
    for l in dead_locations:
        advice, severity = issue_meta.get('dead_location', ("", "low"))
        data["issues"].append({
            "type": "dead_location",
            "severity": severity,
            "message": f"server: {l.get('server', {}).get('arg','')} location: {l.get('location', {}).get('arg','')}",
            "advice": advice
        })
    
    # Подсчитываем статистику
    severity_counts = {"high": 0, "medium": 0, "low": 0}
    for issue in data["issues"]:
        severity = issue.get("severity", "medium")
        if severity in severity_counts:
            severity_counts[severity] += 1
    
    data["summary"] = {
        "total_issues": len(data["issues"]),
        "by_severity": severity_counts
    }
    
    return data


def format_logs_results(
    status_counter,
    path_counter,
    ip_counter,
    user_agent_counter,
    errors: Dict[str, List[str]],
    top: int,
    response_times: Optional[Dict[str, float]] = None,
    anomalies: Optional[List[Dict[str, Any]]] = None
) -> Dict[str, Any]:
    """
    Форматирует результаты команды logs для экспорта.
    
    Args:
        status_counter: Счетчик статусов
        path_counter: Счетчик путей
        ip_counter: Счетчик IP
        user_agent_counter: Счетчик User-Agent
        errors: Словарь ошибок по статусам
        top: Количество топ-значений
        
    Returns:
        Словарь с данными для экспорта
    """
    data = {
        "timestamp": datetime.now().isoformat(),
        "top_statuses": [{"status": status, "count": count} for status, count in status_counter.most_common(top)],
        "top_paths": [{"path": path, "count": count} for path, count in path_counter.most_common(top)],
        "top_ips": [{"ip": ip, "count": count} for ip, count in ip_counter.most_common(top)],
        "errors": {}
    }
    
    if user_agent_counter:
        data["top_user_agents"] = [{"user_agent": ua, "count": count} for ua, count in user_agent_counter.most_common(top)]
    
    for status, paths in errors.items():
        data["errors"][status] = {
            "count": len(paths),
            "unique_paths": list(set(paths))[:top]
        }
    
    data["summary"] = {
        "total_requests": sum(status_counter.values()),
        "unique_paths": len(path_counter),
        "unique_ips": len(ip_counter),
        "error_requests": sum(len(paths) for paths in errors.values())
    }
    
    if response_times:
        data["response_times"] = response_times
    
    if anomalies:
        data["anomalies"] = anomalies
    
    return data

