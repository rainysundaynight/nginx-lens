import sys
import typer
from rich.console import Console
from rich.panel import Panel
from analyzer.route import find_route
from parser.nginx_parser import parse_nginx_config
import glob
import os
from config.config_loader import get_config

app = typer.Typer(help="Показывает, какой server/location обслуживает указанный URL. По умолчанию ищет во всех .conf в /etc/nginx/. Для кастомного пути используйте -c/--config.")
console = Console()

def route(
    url: str = typer.Argument(..., help="URL для маршрутизации (например, http://host/path)"),
    config_path: str = typer.Option(None, "-c", "--config", help="Путь к кастомному nginx.conf (если не указан — поиск по всем .conf в /etc/nginx)")
):
    """
    Показывает, какой server/location обслуживает указанный URL.

    По умолчанию ищет во всех .conf в /etc/nginx/.
    Для кастомного пути используйте опцию -c/--config.

    Примеры:
        nginx-lens route http://example.com/api/v1
        nginx-lens route -c /etc/nginx/nginx.conf http://example.com/api/v1
    """
    configs = []
    if config_path:
        configs = [config_path]
    else:
        # Сначала пробуем получить из конфига
        config = get_config()
        nginx_config_path = config.get_nginx_config_path()
        if nginx_config_path:
            configs = [nginx_config_path]
        else:
            # Если не найден в конфиге, ищем все .conf в /etc/nginx
            configs = glob.glob("/etc/nginx/**/*.conf", recursive=True)
            if not configs:
                console.print(Panel("Не найдено ни одного .conf файла. Укажите путь через -c/--config или настройте nginx_config_path в конфиге.", style="red"))
                return
    for conf in configs:
        try:
            tree = parse_nginx_config(conf)
        except FileNotFoundError:
            console.print(f"[red]Файл {conf} не найден. Проверьте путь к конфигу.[/red]")
            continue
        except Exception as e:
            console.print(f"[red]Ошибка при разборе {conf}: {e}[/red]")
            continue
        res = find_route(tree, url)
        if res:
            server = res['server']
            location = res['location']
            proxy_pass = res['proxy_pass']
            text = f"[bold]Config:[/bold] {conf}\n"
            if server:
                text += f"[bold]Server:[/bold] {server.get('arg','') or '[no arg]'}"
                if server.get('__file__'):
                    text += f" ([dim]{server.get('__file__')}[/dim])"
                text += "\n"
            if location:
                text += f"[bold]Location:[/bold] {location.get('arg','')}"
                if location.get('__file__'):
                    text += f" ([dim]{location.get('__file__')}[/dim])"
                text += "\n"
            if proxy_pass:
                text += f"[bold]proxy_pass:[/bold] {proxy_pass}\n"
            console.print(Panel(text, title="Route", style="green"))
            return
    search_dir = os.path.dirname(configs[0]) if configs else '/etc/nginx'
    console.print(Panel(f"Маршрут для {url} не найден ни в одном .conf в {search_dir}", style="yellow")) 