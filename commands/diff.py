import sys
import typer
from rich.console import Console
from rich.table import Table
from analyzer.diff import diff_trees
from parser.nginx_parser import parse_nginx_config
import difflib

app = typer.Typer()
console = Console()

def diff(
    config1: str = typer.Argument(..., help="Первый nginx.conf"),
    config2: str = typer.Argument(..., help="Второй nginx.conf")
):
    """
    Сравнивает две конфигурации Nginx и выводит отличия построчно side-by-side с подсветкой.

    Пример:
        nginx-lens diff /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak
    """
    try:
        with open(config1) as f1, open(config2) as f2:
            lines1 = f1.readlines()
            lines2 = f2.readlines()
    except FileNotFoundError as e:
        console.print(f"[red]Файл {e.filename} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при чтении файлов: {e}[/red]")
        sys.exit(1)
    maxlen = max(len(lines1), len(lines2))
    # Выравниваем длины
    lines1 += [''] * (maxlen - len(lines1))
    lines2 += [''] * (maxlen - len(lines2))
    table = Table(show_header=True, header_style="bold blue")
    table.add_column("№", style="dim", width=4)
    table.add_column("Config 1", style="white")
    table.add_column("№", style="dim", width=4)
    table.add_column("Config 2", style="white")
    for i in range(maxlen):
        l1 = lines1[i].rstrip('\n')
        l2 = lines2[i].rstrip('\n')
        n1 = str(i+1) if l1 else ''
        n2 = str(i+1) if l2 else ''
        if l1 == l2:
            table.add_row(n1, l1, n2, l2)
        else:
            style1 = "red" if l1 else "on red"
            style2 = "green" if l2 else "on green"
            table.add_row(f"[bold]{n1}[/bold]" if l1 else n1, f"[{style1}]{l1}[/{style1}]" if l1 or l2 else '', f"[bold]{n2}[/bold]" if l2 else n2, f"[{style2}]{l2}[/{style2}]" if l1 or l2 else '')
    console.print(table) 