# nginx_lens/commands/health.py

import typer
from rich.console import Console
from rich.table import Table
from parser.nginx_parser import parse_nginx_config
from upstream_checker.checker import check_upstreams

console = Console()
app = typer.Typer()


@app.command()
def health(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    timeout: float = typer.Option(2.0, help="Таймаут проверки (сек)"),
    retries: int = typer.Option(1, help="Количество попыток"),
    mode: str = typer.Option("tcp", help="Режим проверки: tcp или http", case_sensitive=False),
):
    """
    Проверяет доступность upstream-серверов, определённых в nginx.conf.
    """
    try:
        tree = parse_nginx_config(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден.[/red]")
        raise typer.Exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        raise typer.Exit(1)

    upstreams = tree.get_upstreams()
    results = check_upstreams(upstreams, timeout=timeout, retries=retries, mode=mode.lower())

    table = Table(show_header=True, header_style="bold blue")
    table.add_column("upstream_name")
    table.add_column("address")
    table.add_column("status")

    for name, servers in results.items():
        for srv in servers:
            status = "Healthy" if srv["healthy"] else "Unhealthy"
            color = "green" if srv["healthy"] else "red"
            table.add_row(name, srv["address"], f"[{color}]{status}[/{color}]")

    console.print(table)
