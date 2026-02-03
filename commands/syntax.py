import typer
from rich.console import Console
from rich.table import Table
import subprocess
import os
import re
from config.config_loader import get_config

app = typer.Typer(help="Проверка синтаксиса nginx-конфига через nginx -t с подсветкой ошибок.")
console = Console()

ERRORS_RE = re.compile(r'in (.+?):(\d+)(?:\s*\n)?(.+?)(?=\nin |$)', re.DOTALL)

def syntax(
    config_path: str = typer.Option(None, "-c", "--config", help="Путь к кастомному nginx.conf"),
    nginx_path: str = typer.Option("nginx", help="Путь к бинарю nginx (по умолчанию 'nginx')")
):
    """
    Проверяет синтаксис nginx-конфига через nginx -t.

    В случае ошибки показывает место в виде таблицы с контекстом.

    Пример:
        nginx-lens syntax -c ./mynginx.conf
        nginx-lens syntax
    """
    if not config_path:
        # Сначала пробуем получить из конфига
        config = get_config()
        config_path = config.get_nginx_config_path()
        
        # Если не найден в конфиге, пробуем автопоиск
        if not config_path:
            candidates = [
                "/etc/nginx/nginx.conf",
                "/usr/local/etc/nginx/nginx.conf",
                "./nginx.conf"
            ]
            config_path = next((p for p in candidates if os.path.isfile(p)), None)
        
        if not config_path:
            console.print("[red]Не удалось найти nginx.conf.[/red]")
            console.print("[yellow]Укажите путь через -c или настройте nginx_config_path в конфиге.[/yellow]")
            return
    if not os.path.isfile(config_path):
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        return
    
    cmd = [nginx_path, "-t", "-c", os.path.abspath(config_path)]
    if hasattr(os, 'geteuid') and os.geteuid() != 0:
        cmd = ["sudo"] + cmd
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=False)
        if result.returncode == 0:
            console.print("[green]Синтаксис nginx-конфига корректен[/green]")
            return
        else:
            console.print("[red]Ошибка синтаксиса![/red]")
            console.print(result.stdout)
            console.print(result.stderr)
        # Парсим все ошибки
        err = result.stderr or result.stdout
        errors = list(ERRORS_RE.finditer(err))
        if not errors:
            console.print("[red]Не удалось определить файл и строку ошибки[/red]")
            return
        table = Table(title="Ошибки синтаксиса", show_header=True, header_style="bold red")
        table.add_column("config_file")
        table.add_column("issue_message")
        table.add_column("context")
        for m in errors:
            file, line, msg = m.group(1), int(m.group(2)), m.group(3).strip().split('\n')[0]
            # Читаем контекст
            context_lines = []
            try:
                with open(file) as f:
                    lines = f.readlines()
                start = max(0, line-3)
                end = min(len(lines), line+2)
                for i in range(start, end):
                    mark = "->" if i+1 == line else "  "
                    context_lines.append(f"{mark} {lines[i].rstrip()}")
            except Exception:
                context_lines = []
            table.add_row(file, msg, '\n'.join(context_lines))
        console.print(table)
    except FileNotFoundError:
        console.print(f"[red]Не найден бинарь nginx: {nginx_path}[/red]") 