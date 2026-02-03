import sys
from typing import Optional, Dict, Any, List
import typer
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich import box
from collections import Counter, defaultdict

from parser.nginx_parser import parse_nginx_config

app = typer.Typer()
console = Console()


def _count_blocks(tree, block_type: str) -> int:
    """
    Подсчитывает количество блоков определенного типа в дереве конфигурации.
    
    Args:
        tree: Дерево конфигурации Nginx
        block_type: Тип блока (например, 'server', 'location', 'upstream')
        
    Returns:
        Количество блоков
    """
    # Для upstream используем get_upstreams()
    if block_type == 'upstream':
        if hasattr(tree, 'get_upstreams'):
            upstreams = tree.get_upstreams()
            return len(upstreams)
        return 0
    
    count = 0
    
    def traverse(directives):
        nonlocal count
        for directive in directives:
            if directive.get('block') == block_type:
                count += 1
            if 'directives' in directive:
                traverse(directive['directives'])
    
    if hasattr(tree, 'directives'):
        traverse(tree.directives)
    
    return count


def _count_directives(tree) -> Dict[str, int]:
    """
    Подсчитывает количество директив каждого типа в конфигурации.
    
    Args:
        tree: Дерево конфигурации Nginx
        
    Returns:
        Словарь {directive_name: count}
    """
    directive_counts = Counter()
    
    def traverse(directives):
        for directive in directives:
            if 'directive' in directive:
                directive_counts[directive['directive']] += 1
            if 'directives' in directive:
                traverse(directive['directives'])
    
    if hasattr(tree, 'directives'):
        traverse(tree.directives)
    
    return dict(directive_counts)


def _get_upstream_servers_count(tree) -> Dict[str, int]:
    """
    Подсчитывает количество серверов в каждом upstream.
    
    Args:
        tree: Дерево конфигурации Nginx
        
    Returns:
        Словарь {upstream_name: server_count}
    """
    upstreams = tree.get_upstreams() if hasattr(tree, 'get_upstreams') else {}
    return {name: len(servers) for name, servers in upstreams.items()}


def _collect_metrics(config_path: str) -> Dict[str, Any]:
    """
    Собирает метрики о конфигурации Nginx.
    
    Args:
        config_path: Путь к конфигурационному файлу
        
    Returns:
        Словарь с метриками
    """
    try:
        tree = parse_nginx_config(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        sys.exit(1)
    
    # Подсчет блоков
    upstream_count = _count_blocks(tree, 'upstream')
    server_count = _count_blocks(tree, 'server')
    location_count = _count_blocks(tree, 'location')
    
    # Подсчет директив
    directive_counts = _count_directives(tree)
    
    # Подсчет серверов в upstream
    upstream_servers = _get_upstream_servers_count(tree)
    total_upstream_servers = sum(upstream_servers.values())
    
    # Статистика по server блокам
    server_stats = _get_server_stats(tree)
    
    metrics = {
        "config_path": config_path,
        "blocks": {
            "upstream": upstream_count,
            "server": server_count,
            "location": location_count,
        },
        "upstream_servers": {
            "total": total_upstream_servers,
            "by_upstream": upstream_servers,
        },
        "directives": directive_counts,
        "server_stats": server_stats,
    }
    
    return metrics


def _get_server_stats(tree) -> Dict[str, Any]:
    """
    Собирает статистику по server блокам.
    
    Args:
        tree: Дерево конфигурации Nginx
        
    Returns:
        Словарь со статистикой
    """
    stats = {
        "with_ssl": 0,
        "with_http2": 0,
        "listen_ports": Counter(),
        "server_names": [],
    }
    
    def traverse(directives):
        current_server = None
        for directive in directives:
            if directive.get('block') == 'server':
                current_server = {
                    'has_ssl': False,
                    'has_http2': False,
                    'listen_ports': [],
                    'server_names': [],
                }
            elif current_server is not None:
                if directive.get('directive') == 'listen':
                    args = directive.get('args', '')
                    if 'ssl' in args:
                        current_server['has_ssl'] = True
                    if 'http2' in args:
                        current_server['has_http2'] = True
                    # Извлекаем порт
                    port = _extract_port(args)
                    if port:
                        current_server['listen_ports'].append(port)
                        stats["listen_ports"][port] += 1
                elif directive.get('directive') == 'server_name':
                    server_names = directive.get('args', '').split()
                    current_server['server_names'].extend(server_names)
                    stats["server_names"].extend(server_names)
            
            if 'directives' in directive:
                traverse(directive['directives'])
    
    if hasattr(tree, 'directives'):
        traverse(tree.directives)
    
    # Подсчитываем серверы с SSL и HTTP2
    def count_servers(directives):
        for directive in directives:
            if directive.get('block') == 'server':
                has_ssl = False
                has_http2 = False
                for d in directive.get('directives', []):
                    if d.get('directive') == 'listen':
                        args = d.get('args', '')
                        if 'ssl' in args:
                            has_ssl = True
                        if 'http2' in args:
                            has_http2 = True
                if has_ssl:
                    stats["with_ssl"] += 1
                if has_http2:
                    stats["with_http2"] += 1
            if 'directives' in directive:
                count_servers(directive['directives'])
    
    if hasattr(tree, 'directives'):
        count_servers(tree.directives)
    
    stats["listen_ports"] = dict(stats["listen_ports"])
    stats["server_names"] = list(set(stats["server_names"]))
    
    return stats


def _extract_port(listen_args: str) -> Optional[int]:
    """
    Извлекает порт из директивы listen.
    
    Args:
        listen_args: Аргументы директивы listen
        
    Returns:
        Номер порта или None
    """
    import re
    # Ищем порт в формате :80, :443, listen 80, listen 443
    match = re.search(r':(\d+)', listen_args)
    if match:
        return int(match.group(1))
    match = re.search(r'\b(\d+)\b', listen_args)
    if match:
        return int(match.group(1))
    return None


def _export_prometheus(metrics: Dict[str, Any]) -> str:
    """
    Экспортирует метрики в формате Prometheus.
    
    Args:
        metrics: Словарь с метриками
        
    Returns:
        Строка в формате Prometheus
    """
    lines = []
    lines.append("# HELP nginx_lens_upstream_count Total number of upstream blocks")
    lines.append("# TYPE nginx_lens_upstream_count gauge")
    lines.append(f'nginx_lens_upstream_count{{config="{metrics["config_path"]}"}} {metrics["blocks"]["upstream"]}')
    
    lines.append("# HELP nginx_lens_server_count Total number of server blocks")
    lines.append("# TYPE nginx_lens_server_count gauge")
    lines.append(f'nginx_lens_server_count{{config="{metrics["config_path"]}"}} {metrics["blocks"]["server"]}')
    
    lines.append("# HELP nginx_lens_location_count Total number of location blocks")
    lines.append("# TYPE nginx_lens_location_count gauge")
    lines.append(f'nginx_lens_location_count{{config="{metrics["config_path"]}"}} {metrics["blocks"]["location"]}')
    
    lines.append("# HELP nginx_lens_upstream_servers_total Total number of upstream servers")
    lines.append("# TYPE nginx_lens_upstream_servers_total gauge")
    lines.append(f'nginx_lens_upstream_servers_total{{config="{metrics["config_path"]}"}} {metrics["upstream_servers"]["total"]}')
    
    # Метрики по upstream
    for upstream_name, server_count in metrics["upstream_servers"]["by_upstream"].items():
        lines.append(f'nginx_lens_upstream_servers{{config="{metrics["config_path"]}",upstream="{upstream_name}"}} {server_count}')
    
    # Метрики по директивам (топ-10)
    top_directives = sorted(metrics["directives"].items(), key=lambda x: x[1], reverse=True)[:10]
    for directive, count in top_directives:
        safe_directive = directive.replace('-', '_').replace('.', '_')
        lines.append(f'nginx_lens_directive_count{{config="{metrics["config_path"]}",directive="{directive}"}} {count}')
    
    # Метрики по портам
    for port, count in metrics["server_stats"]["listen_ports"].items():
        lines.append(f'nginx_lens_listen_port_count{{config="{metrics["config_path"]}",port="{port}"}} {count}')
    
    # Метрики SSL и HTTP2
    lines.append(f'nginx_lens_servers_with_ssl{{config="{metrics["config_path"]}"}} {metrics["server_stats"]["with_ssl"]}')
    lines.append(f'nginx_lens_servers_with_http2{{config="{metrics["config_path"]}"}} {metrics["server_stats"]["with_http2"]}')
    
    return '\n'.join(lines)


def metrics(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    compare_with: Optional[str] = typer.Option(None, "--compare", "-c", help="Путь к другому конфигу для сравнения"),
    prometheus: bool = typer.Option(False, "--prometheus", "-p", help="Экспортировать в формате Prometheus"),
    json: bool = typer.Option(False, "--json", help="Экспортировать результаты в JSON"),
    yaml: bool = typer.Option(False, "--yaml", help="Экспортировать результаты в YAML"),
):
    """
    Собирает метрики о конфигурации Nginx.
    
    Показывает:
      - Количество upstream, server, location блоков
      - Количество серверов в каждом upstream
      - Статистику по директивам
      - Статистику по server блокам (SSL, HTTP2, порты)
    
    Поддерживает сравнение метрик между двумя версиями конфига.
    Экспорт в Prometheus format для интеграции с мониторингом.

    Пример:
        nginx-lens metrics /etc/nginx/nginx.conf
        nginx-lens metrics /etc/nginx/nginx.conf --prometheus
        nginx-lens metrics /etc/nginx/nginx.conf --compare /etc/nginx/nginx.conf.old
    """
    # Собираем метрики
    metrics_data = _collect_metrics(config_path)
    
    # Если нужно сравнить с другим конфигом
    if compare_with:
        compare_metrics = _collect_metrics(compare_with)
        metrics_data["comparison"] = _compare_metrics(metrics_data, compare_metrics)
    
    # Экспорт в Prometheus
    if prometheus:
        prometheus_output = _export_prometheus(metrics_data)
        console.print(prometheus_output)
        sys.exit(0)
    
    # Экспорт в JSON/YAML
    if json or yaml:
        from exporter.json_yaml import print_export
        export_data = {
            "timestamp": __import__('datetime').datetime.now().isoformat(),
            "metrics": metrics_data
        }
        format_type = 'json' if json else 'yaml'
        print_export(export_data, format_type)
        sys.exit(0)
    
    # Обычный вывод
    _print_metrics_table(metrics_data, compare_with is not None)
    sys.exit(0)


def _compare_metrics(metrics1: Dict[str, Any], metrics2: Dict[str, Any]) -> Dict[str, Any]:
    """
    Сравнивает метрики двух конфигураций.
    
    Args:
        metrics1: Метрики первой конфигурации
        metrics2: Метрики второй конфигурации
        
    Returns:
        Словарь с различиями
    """
    comparison = {
        "blocks": {},
        "upstream_servers": {},
        "directives": {},
    }
    
    # Сравнение блоков
    for block_type in ["upstream", "server", "location"]:
        count1 = metrics1["blocks"][block_type]
        count2 = metrics2["blocks"][block_type]
        diff = count1 - count2
        comparison["blocks"][block_type] = {
            "current": count1,
            "previous": count2,
            "diff": diff,
            "percent_change": (diff / count2 * 100) if count2 > 0 else 0
        }
    
    # Сравнение upstream серверов
    total1 = metrics1["upstream_servers"]["total"]
    total2 = metrics2["upstream_servers"]["total"]
    comparison["upstream_servers"]["total"] = {
        "current": total1,
        "previous": total2,
        "diff": total1 - total2,
        "percent_change": ((total1 - total2) / total2 * 100) if total2 > 0 else 0
    }
    
    # Сравнение директив (топ-10)
    all_directives = set(metrics1["directives"].keys()) | set(metrics2["directives"].keys())
    top_directives = sorted(all_directives, key=lambda d: metrics1["directives"].get(d, 0) + metrics2["directives"].get(d, 0), reverse=True)[:10]
    
    for directive in top_directives:
        count1 = metrics1["directives"].get(directive, 0)
        count2 = metrics2["directives"].get(directive, 0)
        diff = count1 - count2
        comparison["directives"][directive] = {
            "current": count1,
            "previous": count2,
            "diff": diff,
            "percent_change": (diff / count2 * 100) if count2 > 0 else 0
        }
    
    return comparison


def _print_metrics_table(metrics_data: Dict[str, Any], show_comparison: bool = False):
    """
    Выводит метрики в виде таблицы.
    
    Args:
        metrics_data: Словарь с метриками
        show_comparison: Показывать ли сравнение
    """
    console.print(Panel("[bold blue]Метрики конфигурации Nginx[/bold blue]", box=box.ROUNDED))
    
    # Таблица блоков
    table = Table(title="Блоки", show_header=True, header_style="bold blue")
    table.add_column("Тип блока")
    table.add_column("Количество")
    
    if show_comparison:
        table.add_column("Предыдущее")
        table.add_column("Изменение")
    
    blocks = metrics_data["blocks"]
    comparison = metrics_data.get("comparison", {})
    
    for block_type in ["upstream", "server", "location"]:
        count = blocks[block_type]
        if show_comparison and "blocks" in comparison:
            prev = comparison["blocks"].get(block_type, {}).get("previous", 0)
            diff = comparison["blocks"].get(block_type, {}).get("diff", 0)
            color = "green" if diff >= 0 else "red"
            table.add_row(
                block_type,
                str(count),
                str(prev),
                f"[{color}]{diff:+d}[/{color}]"
            )
        else:
            table.add_row(block_type, str(count))
    
    console.print(table)
    
    # Таблица upstream серверов
    if metrics_data["upstream_servers"]["by_upstream"]:
        table = Table(title="Upstream серверы", show_header=True, header_style="bold blue")
        table.add_column("Upstream")
        table.add_column("Количество серверов")
        
        for upstream_name, server_count in sorted(metrics_data["upstream_servers"]["by_upstream"].items()):
            table.add_row(upstream_name, str(server_count))
        
        table.add_row("[bold]Всего[/bold]", f"[bold]{metrics_data['upstream_servers']['total']}[/bold]")
        console.print(table)
    
    # Таблица топ-директив
    top_directives = sorted(metrics_data["directives"].items(), key=lambda x: x[1], reverse=True)[:10]
    if top_directives:
        table = Table(title="Топ-10 директив", show_header=True, header_style="bold blue")
        table.add_column("Директива")
        table.add_column("Количество")
        
        if show_comparison and "directives" in comparison:
            table.add_column("Предыдущее")
            table.add_column("Изменение")
        
        for directive, count in top_directives:
            if show_comparison and directive in comparison["directives"]:
                comp = comparison["directives"][directive]
                prev = comp["previous"]
                diff = comp["diff"]
                color = "green" if diff >= 0 else "red"
                table.add_row(
                    directive,
                    str(count),
                    str(prev),
                    f"[{color}]{diff:+d}[/{color}]"
                )
            else:
                table.add_row(directive, str(count))
        
        console.print(table)
    
    # Статистика по server блокам
    server_stats = metrics_data["server_stats"]
    if server_stats["listen_ports"] or server_stats["with_ssl"] > 0 or server_stats["with_http2"] > 0:
        table = Table(title="Статистика server блоков", show_header=True, header_style="bold blue")
        table.add_column("Метрика")
        table.add_column("Значение")
        
        table.add_row("С SSL", str(server_stats["with_ssl"]))
        table.add_row("С HTTP/2", str(server_stats["with_http2"]))
        
        if server_stats["listen_ports"]:
            top_ports = sorted(server_stats["listen_ports"].items(), key=lambda x: x[1], reverse=True)[:5]
            for port, count in top_ports:
                table.add_row(f"Порт {port}", str(count))
        
        console.print(table)

