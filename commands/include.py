import sys
import typer
from rich.console import Console
from rich.tree import Tree
from rich.table import Table
from analyzer.include import build_include_tree, find_include_cycles, find_include_shadowing

app = typer.Typer()
console = Console()

def include_tree(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    directive: str = typer.Option(None, help="Показать shadowing для директивы (например, server_name)")
):
    """
    Показывает дерево include-ов, циклы и shadowing директив.

    Пример:
        nginx-lens include-tree /etc/nginx/nginx.conf
        nginx-lens include-tree /etc/nginx/nginx.conf --directive server_name
    """
    try:
        tree = build_include_tree(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        sys.exit(1)
    rich_tree = Tree(f"[bold blue]{config_path}[/bold blue]")
    def _add(node, t):
        for k, v in t.items():
            if v == 'cycle':
                node.add(f"[red]{k} (cycle)[/red]")
            elif v == 'not_found':
                node.add(f"[yellow]{k} (not found)[/yellow]")
            elif isinstance(v, list):
                sub = node.add(f"{k}")
                for sub_t in v:
                    if isinstance(sub_t, dict):
                        _add(sub, sub_t)
    _add(rich_tree, tree)
    console.print(rich_tree)
    # Циклы
    cycles = find_include_cycles(tree)
    if cycles:
        console.print("[red]Обнаружены циклы include-ов:[/red]")
        for c in cycles:
            console.print(" -> ".join(c))
    # Shadowing
    if directive:
        shadow = find_include_shadowing(tree, directive)
        if shadow:
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("file")
            table.add_column("value")
            for s in shadow:
                table.add_row(s['file'], s['value'])
            console.print(table) 