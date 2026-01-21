import sys
import typer
from rich.console import Console
from rich.table import Table
from upstream_checker.checker import resolve_upstreams
from parser.nginx_parser import parse_nginx_config

app = typer.Typer()
console = Console()

def resolve(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
):
    """
    Резолвит DNS имена upstream-серверов в IP-адреса.

    Пример:
        nginx-lens resolve /etc/nginx/nginx.conf
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
    if not upstreams:
        console.print("[yellow]Не найдено ни одного upstream в конфигурации.[/yellow]")
        sys.exit(0)  # Нет upstream - это не ошибка, просто нет чего проверять
    
    results = resolve_upstreams(upstreams)

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

