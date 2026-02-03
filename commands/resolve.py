import sys
from typing import Optional
import typer
from rich.console import Console
from rich.table import Table
from upstream_checker.checker import resolve_upstreams
from upstream_checker.dns_cache import disable_cache, enable_cache, clear_cache
from parser.nginx_parser import parse_nginx_config
from exporter.json_yaml import format_resolve_results, print_export
from config.config_loader import get_config
from utils.progress import ProgressManager

app = typer.Typer()
console = Console()

def resolve(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    max_workers: Optional[int] = typer.Option(None, "--max-workers", "-w", help="Максимальное количество потоков для параллельной обработки"),
    json: bool = typer.Option(False, "--json", help="Экспортировать результаты в JSON"),
    yaml: bool = typer.Option(False, "--yaml", help="Экспортировать результаты в YAML"),
    no_cache: bool = typer.Option(False, "--no-cache", help="Отключить кэширование DNS резолвинга"),
    cache_ttl: Optional[int] = typer.Option(None, "--cache-ttl", help="Время жизни кэша в секундах"),
):
    """
    Резолвит DNS имена upstream-серверов в IP-адреса.
    Использует параллельную обработку для ускорения резолвинга множества upstream серверов.
    Поддерживает кэширование результатов DNS резолвинга для ускорения повторных запусков.

    Пример:
        nginx-lens resolve /etc/nginx/nginx.conf
        nginx-lens resolve /etc/nginx/nginx.conf --max-workers 20
        nginx-lens resolve /etc/nginx/nginx.conf --no-cache
        nginx-lens resolve /etc/nginx/nginx.conf --cache-ttl 600
    """
    exit_code = 0
    
    # Загружаем конфигурацию
    config = get_config()
    defaults = config.get_defaults()
    cache_config = config.get_cache_config()
    
    # Применяем значения из конфига, если не указаны через CLI
    max_workers = max_workers if max_workers is not None else defaults.get("max_workers", 10)
    cache_ttl = cache_ttl if cache_ttl is not None else cache_config.get("ttl", defaults.get("dns_cache_ttl", 300))
    
    # Управление кэшем
    use_cache = not no_cache and cache_config.get("enabled", True)
    if no_cache:
        disable_cache()
    else:
        enable_cache()
    
    try:
        tree = parse_nginx_config(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        sys.exit(1)

    upstreams = tree.get_upstreams()
    if not upstreams:
        if json or yaml:
            export_data = {
                "timestamp": __import__('datetime').datetime.now().isoformat(),
                "upstreams": [],
                "summary": {"total_upstreams": 0, "total_servers": 0}
            }
            format_type = 'json' if json else 'yaml'
            print_export(export_data, format_type)
        else:
            console.print("[yellow]Не найдено ни одного upstream в конфигурации.[/yellow]")
        sys.exit(0)  # Нет upstream - это не ошибка, просто нет чего проверять
    
    # Подсчитываем общее количество серверов для прогресс-бара
    total_servers = sum(len(servers) for servers in upstreams.values())
    
    # Резолвинг с прогресс-баром
    with ProgressManager(description="Резолвинг DNS", show_progress=total_servers > 5) as pm:
        results = resolve_upstreams(upstreams, max_workers=max_workers, use_cache=use_cache, cache_ttl=cache_ttl, progress_manager=pm)

    # Экспорт в JSON/YAML
    if json or yaml:
        export_data = format_resolve_results(results)
        format_type = 'json' if json else 'yaml'
        print_export(export_data, format_type)
        # Exit code остается прежним
        for name, servers in results.items():
            for srv in servers:
                if not srv["resolved"] or any("invalid resolve" in r for r in srv["resolved"]):
                    exit_code = 1
        sys.exit(exit_code)

    # Обычный вывод в таблицу
    table = Table(show_header=True, header_style="bold blue")
    table.add_column("Upstream Name")
    table.add_column("Address")
    table.add_column("Resolved IP")

    for name, servers in results.items():
        for idx, srv in enumerate(servers):
            upstream_name = name if idx == 0 else ""
            resolved_list = srv["resolved"]
            if resolved_list:
                # Показываем все IP-адреса через запятую
                resolved_str = ", ".join(resolved_list)
                # Если есть "invalid resolve", показываем красным, иначе зеленым
                if any("invalid resolve" in r for r in resolved_list):
                    table.add_row(upstream_name, srv["address"], f"[red]{resolved_str}[/red]")
                    exit_code = 1
                else:
                    table.add_row(upstream_name, srv["address"], f"[green]{resolved_str}[/green]")
            else:
                table.add_row(upstream_name, srv["address"], "[red]Failed to resolve[/red]")
                exit_code = 1

    console.print(table)
    sys.exit(exit_code)

