import sys
from typing import Optional
import typer
from rich.console import Console
from rich.tree import Tree as RichTree
from parser.nginx_parser import parse_nginx_config
from config.config_loader import get_config

app = typer.Typer()
console = Console()

def _build_tree(directives, parent):
    for d in directives:
        if 'block' in d:
            label = f"[bold]{d['block']}[/bold] {d.get('arg') or ''}".strip()
            node = parent.add(label)
            if d.get('directives'):
                _build_tree(d['directives'], node)
        elif 'upstream' in d:
            label = f"[bold magenta]upstream[/bold magenta] {d['upstream']}"
            node = parent.add(label)
            for srv in d.get('servers', []):
                node.add(f"[green]server[/green] {srv}")
        elif 'directive' in d:
            parent.add(f"[cyan]{d['directive']}[/cyan] {d.get('args','')}")

def tree(
    config_path: Optional[str] = typer.Argument(None, help="Путь к nginx.conf (если не указан, используется из конфига или автопоиск)"),
    markdown: bool = typer.Option(False, help="Экспортировать в Markdown"),
    html: bool = typer.Option(False, help="Экспортировать в HTML")
):
    """
    Визуализирует структуру nginx.conf в виде дерева.

    Пример:
        nginx-lens tree /etc/nginx/nginx.conf
        nginx-lens tree /etc/nginx/nginx.conf --markdown
        nginx-lens tree /etc/nginx/nginx.conf --html
        nginx-lens tree  # Использует путь из конфига
    """
    # Определяем путь к конфигу
    if not config_path:
        config = get_config()
        config_path = config.get_nginx_config_path()
        if not config_path:
            console.print("[red]Путь к nginx.conf не указан и не найден автоматически.[/red]")
            console.print("[yellow]Укажите путь через аргумент или настройте nginx_config_path в конфиге.[/yellow]")
            sys.exit(1)
    
    try:
        tree_obj = parse_nginx_config(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        sys.exit(1)
    root = RichTree(f"[bold blue]nginx.conf[/bold blue]")
    _build_tree(tree_obj.directives, root)
    if markdown:
        from exporter.markdown import tree_to_markdown
        md = tree_to_markdown(tree_obj.directives)
        console.print(md)
    elif html:
        from exporter.html import tree_to_html
        html_code = tree_to_html(tree_obj.directives)
        console.print(html_code)
    else:
        console.print(root) 