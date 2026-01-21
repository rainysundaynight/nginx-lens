import sys
import typer
from rich.console import Console
from rich.table import Table
from upstream_checker.checker import check_upstreams, resolve_upstreams
from parser.nginx_parser import parse_nginx_config

app = typer.Typer()
console = Console()

def health(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    timeout: float = typer.Option(2.0, help="Таймаут проверки (сек)"),
    retries: int = typer.Option(1, help="Количество попыток"),
    mode: str = typer.Option("tcp", help="Режим проверки: tcp или http", case_sensitive=False),
    resolve: bool = typer.Option(False, "--resolve", "-r", help="Показать резолвленные IP-адреса"),
):
    """
    Проверяет доступность upstream-серверов, определённых в nginx.conf. Выводит таблицу.

    Пример:
        nginx-lens health /etc/nginx/nginx.conf
        nginx-lens health /etc/nginx/nginx.conf --timeout 5 --retries 3 --mode http
        nginx-lens health /etc/nginx/nginx.conf --resolve
    """
    exit_code = 0
    
    try:
        tree = parse_nginx_config(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        sys.exit(1)

    upstreams = tree.get_upstreams()
    results = check_upstreams(upstreams, timeout=timeout, retries=retries, mode=mode.lower())
    
    # Если нужно показать резолвленные IP-адреса
    resolved_info = {}
    if resolve:
        resolved_info = resolve_upstreams(upstreams)

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
