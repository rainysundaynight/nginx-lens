import typer
from rich import box
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
import subprocess
import os
import re
from typing import List
from config.config_loader import get_config

_DEFAULT_NGINX = get_config().get_validate_config().get("nginx_path", "nginx")

app = typer.Typer(help="Проверка синтаксиса nginx-конфига через nginx -t с подсветкой ошибок.")
console = Console()

ERRORS_RE = re.compile(r'in (.+?):(\d+)(?:\s*\n)?(.+?)(?=\nin |$)', re.DOTALL)


def _extract_nginx_warn_lines(output: str) -> List[str]:
    if not output:
        return []
    return [l.strip() for l in output.splitlines() if re.search(r"\[warn\]", l, re.IGNORECASE)]


def syntax(
    config_path: str = typer.Option(None, "-c", "--config", help="Путь к кастомному nginx.conf"),
    nginx_path: str = typer.Option(
        _DEFAULT_NGINX,
        help="Путь к бинарю nginx (по умолчанию: [validate.nginx_path] в конфиге, иначе 'nginx')",
    ),
    skip_warns: bool = typer.Option(
        False,
        "--skip-warns",
        help="Игнорировать предупреждения [warn] от nginx (при exit 0, без сбоя проверки).",
    ),
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
            raise typer.Exit(1)
    if not os.path.isfile(config_path):
        console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
        raise typer.Exit(1)
    
    cmd = [nginx_path, "-t", "-c", os.path.abspath(config_path)]
    if hasattr(os, 'geteuid') and os.geteuid() != 0:
        cmd = ["sudo"] + cmd
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=False)
        combined = ((result.stdout or "") + "\n" + (result.stderr or "")).strip()
        warn_lines = _extract_nginx_warn_lines(combined)

        if result.returncode == 0:
            if warn_lines and not skip_warns:
                wtable = Table(
                    title="[bold yellow]Предупреждения nginx[/bold yellow] [dim]([warn])[/dim]",
                    show_header=True,
                    header_style="bold yellow",
                    box=box.ROUNDED,
                    border_style="yellow",
                    show_lines=True,
                )
                wtable.add_column("Сообщение", overflow="fold")
                for w in warn_lines:
                    wtable.add_row(w)
                console.print(wtable)
                console.print(
                    Panel(
                        "[red]Проверка не пройдена: есть предупреждения [warn]. "
                        "Запустите с [bold]--skip-warns[/bold], если их можно игнорировать.[/red]",
                        box=box.ROUNDED,
                        border_style="red",
                    )
                )
                raise typer.Exit(1)
            if warn_lines and skip_warns:
                wbody = "\n".join(f"[yellow]{w}[/yellow]" for w in warn_lines)
                console.print(
                    Panel(
                        wbody
                        + "\n\n[green]Синтаксис корректен; предупреждения пропущены ([bold]--skip-warns[/bold]).[/green]",
                        title="[dim]Предупреждения (пропущены)[/dim]",
                        box=box.ROUNDED,
                        border_style="yellow",
                    )
                )
                raise typer.Exit(0)
            console.print(
                Panel(
                    f"[green bold]Синтаксис корректен[/green bold]\n[dim]{os.path.abspath(config_path)}[/dim]",
                    title="[bold]nginx -t[/bold]",
                    box=box.ROUNDED,
                    border_style="green",
                )
            )
            raise typer.Exit(0)

        console.print("[red]Ошибка синтаксиса![/red]")
        if result.stdout:
            console.print(result.stdout)
        if result.stderr:
            console.print(result.stderr)
        # Парсим все ошибки
        err = result.stderr or result.stdout
        errors = list(ERRORS_RE.finditer(err))
        if not errors:
            console.print("[red]Не удалось определить файл и строку ошибки[/red]")
            raise typer.Exit(1)
        table = Table(
            title="[bold red]Ошибки синтаксиса[/bold red]",
            show_header=True,
            header_style="bold red",
            box=box.ROUNDED,
            border_style="red",
            show_lines=True,
        )
        table.add_column("Файл", overflow="fold", style="bright_red")
        table.add_column("Сообщение", overflow="fold")
        table.add_column("Контекст", overflow="fold", style="dim")
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
        raise typer.Exit(1)
    except FileNotFoundError:
        console.print(f"[red]Не найден бинарь nginx: {nginx_path}[/red]")
        raise typer.Exit(1)