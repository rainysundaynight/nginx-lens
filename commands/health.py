import sys
from typing import Optional
import typer
from rich.console import Console
from rich.table import Table
from upstream_checker.checker import check_upstreams, resolve_upstreams
from upstream_checker.dns_cache import disable_cache, enable_cache
from parser.nginx_parser import parse_nginx_config
from exporter.json_yaml import format_health_results, print_export
from config.config_loader import get_config
from utils.progress import ProgressManager

app = typer.Typer()
console = Console()

def health(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    timeout: Optional[float] = typer.Option(None, help="Таймаут проверки (сек)"),
    retries: Optional[int] = typer.Option(None, help="Количество попыток"),
    mode: Optional[str] = typer.Option(None, help="Режим проверки: tcp или http", case_sensitive=False),
    resolve: bool = typer.Option(False, "--resolve", "-r", help="Показать резолвленные IP-адреса"),
    max_workers: Optional[int] = typer.Option(None, "--max-workers", "-w", help="Максимальное количество потоков для параллельной обработки"),
    json: bool = typer.Option(False, "--json", help="Экспортировать результаты в JSON"),
    yaml: bool = typer.Option(False, "--yaml", help="Экспортировать результаты в YAML"),
    no_cache: bool = typer.Option(False, "--no-cache", help="Отключить кэширование DNS резолвинга"),
    cache_ttl: Optional[int] = typer.Option(None, "--cache-ttl", help="Время жизни кэша в секундах"),
):
    """
    Проверяет доступность upstream-серверов, определённых в nginx.conf. Выводит таблицу.
    Использует параллельную обработку для ускорения проверки множества upstream серверов.
    Поддерживает кэширование результатов DNS резолвинга для ускорения повторных запусков.

    Пример:
        nginx-lens health /etc/nginx/nginx.conf
        nginx-lens health /etc/nginx/nginx.conf --timeout 5 --retries 3 --mode http
        nginx-lens health /etc/nginx/nginx.conf --resolve
        nginx-lens health /etc/nginx/nginx.conf --max-workers 20
        nginx-lens health /etc/nginx/nginx.conf --resolve --no-cache
        nginx-lens health /etc/nginx/nginx.conf --resolve --cache-ttl 600
    """
    exit_code = 0
    
    # Загружаем конфигурацию
    config = get_config()
    defaults = config.get_defaults()
    cache_config = config.get_cache_config()
    
    # Применяем значения из конфига, если не указаны через CLI
    timeout = timeout if timeout is not None else defaults.get("timeout", 2.0)
    retries = retries if retries is not None else defaults.get("retries", 1)
    mode = mode if mode is not None else defaults.get("mode", "tcp")
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
    
    # Подсчитываем общее количество серверов для прогресс-бара
    total_servers = sum(len(servers) for servers in upstreams.values())
    
    # Проверка upstream с прогресс-баром
    with ProgressManager(description="Проверка upstream серверов", show_progress=total_servers > 5) as pm:
        results = check_upstreams(upstreams, timeout=timeout, retries=retries, mode=mode.lower(), max_workers=max_workers, progress_manager=pm)
    
    # Если нужно показать резолвленные IP-адреса
    resolved_info = {}
    if resolve:
        with ProgressManager(description="Резолвинг DNS", show_progress=total_servers > 5) as pm:
            resolved_info = resolve_upstreams(upstreams, max_workers=max_workers, use_cache=use_cache, cache_ttl=cache_ttl, progress_manager=pm)

    # Экспорт в JSON/YAML
    if json or yaml:
        export_data = format_health_results(results, resolved_info if resolve else None)
        format_type = 'json' if json else 'yaml'
        print_export(export_data, format_type)
        # Exit code остается прежним
        for name, servers in results.items():
            for srv in servers:
                if not srv["healthy"]:
                    exit_code = 1
                if resolve and name in resolved_info:
                    for resolved_srv in resolved_info[name]:
                        if resolved_srv["address"] == srv["address"]:
                            if any("invalid resolve" in r for r in resolved_srv["resolved"]):
                                exit_code = 1
                            break
        sys.exit(exit_code)

    # Обычный вывод в таблицу
    table = Table(show_header=True, header_style="bold blue")
    table.add_column("Address")
    table.add_column("Status")
    if resolve:
        table.add_column("Resolved IP")

    for name, servers in results.items():
        for srv in servers:
            status = "Healthy" if srv["healthy"] else "Unhealthy"
            color = "green" if srv["healthy"] else "red"
            
            # Проверяем статус здоровья
            if not srv["healthy"]:
                exit_code = 1
            
            if resolve:
                resolved_list = []
                if name in resolved_info:
                    for resolved_srv in resolved_info[name]:
                        if resolved_srv["address"] == srv["address"]:
                            resolved_list = resolved_srv["resolved"]
                            break
                
                if resolved_list:
                    # Показываем все IP-адреса через запятую
                    resolved_str = ", ".join(resolved_list)
                    # Если есть "invalid resolve", показываем красным, иначе зеленым
                    if any("invalid resolve" in r for r in resolved_list):
                        table.add_row(srv["address"], f"[{color}]{status}[/{color}]", f"[red]{resolved_str}[/red]")
                        exit_code = 1
                    else:
                        table.add_row(srv["address"], f"[{color}]{status}[/{color}]", f"[green]{resolved_str}[/green]")
                else:
                    table.add_row(srv["address"], f"[{color}]{status}[/{color}]", "[yellow]Failed to resolve[/yellow]")
                    exit_code = 1
            else:
                table.add_row(srv["address"], f"[{color}]{status}[/{color}]")

    console.print(table)
    sys.exit(exit_code)
