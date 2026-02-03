import sys
from typing import Optional
import typer
from rich.console import Console
from parser.nginx_parser import parse_nginx_config
from exporter.graph import tree_to_dot, tree_to_mermaid
from rich.text import Text
import os
from config.config_loader import get_config

app = typer.Typer()
console = Console()

def graph(
    config_path: Optional[str] = typer.Argument(None, help="Путь к nginx.conf (если не указан, используется из конфига или автопоиск)")
):
    """
    Показывает все возможные маршруты nginx в виде цепочек server → location → proxy_pass → upstream → server.

    Пример:
        nginx-lens graph /etc/nginx/nginx.conf
        nginx-lens graph  # Использует путь из конфига
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
        tree = parse_nginx_config(config_path)
    except FileNotFoundError:
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
        sys.exit(1)
    routes = []
    # Для каждого server/location строим маршрут
    def walk(d, chain, upstreams):
        if d.get('block') == 'server':
            for sub in d.get('directives', []):
                walk(sub, chain + [('server', None)], upstreams)
        elif d.get('block') == 'location':
            loc = d.get('arg','')
            for sub in d.get('directives', []):
                walk(sub, chain + [('location', loc)], upstreams)
        elif d.get('directive') == 'proxy_pass':
            val = d.get('args','')
            # ищем, есть ли такой upstream
            up_name = None
            if val.startswith('http://') or val.startswith('https://'):
                up = val.split('://',1)[1].split('/',1)[0]
                if up in upstreams:
                    up_name = up
            if up_name:
                for srv in upstreams[up_name]:
                    routes.append(chain + [('proxy_pass', val), ('upstream', up_name), ('upstream_server', srv)])
            else:
                routes.append(chain + [('proxy_pass', val)])
        elif d.get('upstream'):
            # собираем upstream-ы
            upstreams[d['upstream']] = d.get('servers',[])
        # рекурсивно по всем директивам
        for sub in d.get('directives', []):
            walk(sub, chain, upstreams)
    # Собираем upstream-ы
    upstreams = {}
    for d in tree.directives:
        if d.get('upstream'):
            upstreams[d['upstream']] = d.get('servers',[])
    # Строим маршруты
    for d in tree.directives:
        walk(d, [], upstreams)
    if not routes:
        console.print("[yellow]Не найдено ни одного маршрута[/yellow]")
        return
    main_file = os.path.abspath(config_path)
    # Красивый вывод
    seen = set()
    for route in routes:
        if not route or route[0][0] != 'server':
            continue
        key = tuple(route)
        if key in seen:
            continue
        seen.add(key)
        t = Text()
        # Получаем label для server
        server_val = route[0][1]
        server_block = None
        for d in tree.directives:
            if d.get('block') == 'server':
                server_block = d
                break
        label = get_server_label(server_block) if server_block else 'default'
        server_file = server_block.get('__file__') if server_block else None
        t.append(f"[", style="white")
        t.append(f"server: {label}", style="bold blue")
        if server_file and os.path.abspath(server_file) != main_file:
            t.append(f" ({server_file})", style="grey50")
        t.append("]", style="white")
        for i, (typ, val) in enumerate(route[1:]):
            block = None
            if typ == 'location' and server_block:
                # Найти location-блок внутри server
                for sub in server_block.get('directives', []):
                    if sub.get('block') == 'location' and sub.get('arg') == val:
                        block = sub
                        break
            elif typ == 'upstream':
                # Найти upstream-блок
                for d in tree.directives:
                    if d.get('upstream') == val:
                        block = d
                        break
            if typ == 'location':
                t.append(f" -> [", style="white")
                t.append(f"location: {val}", style="yellow")
                if block and block.get('__file__') and os.path.abspath(block['__file__']) != main_file:
                    t.append(f" ({block['__file__']})", style="grey50")
                t.append("]", style="white")
            elif typ == 'proxy_pass':
                t.append(f" -> proxy_pass: {val}", style="green")
            elif typ == 'upstream':
                t.append(f" -> [", style="white")
                t.append(f"upstream: {val}", style="magenta")
                if block and block.get('__file__') and os.path.abspath(block['__file__']) != main_file:
                    t.append(f" ({block['__file__']})", style="grey50")
                t.append("]", style="white")
            elif typ == 'upstream_server':
                t.append(f" -> [", style="white")
                t.append(f"server: {val}", style="grey50")
                t.append("]", style="white")
        console.print(t)

def get_server_label(server_block):
    arg = server_block.get('arg')
    if arg and arg != '[no arg]':
        return arg
    # Ищем server_name
    names = []
    listens = []
    for sub in server_block.get('directives', []):
        if sub.get('directive') == 'server_name':
            names += sub.get('args', '').split()
        if sub.get('directive') == 'listen':
            listens.append(sub.get('args', ''))
    if names:
        return ' '.join(names)
    if listens:
        return ','.join(listens)
    return 'default'

if __name__ == "__main__":
    app() 