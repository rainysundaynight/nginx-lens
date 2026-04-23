"""
Команда: показать активный конфиг и путь к файлу.
"""
import os
import yaml
import typer
from rich import box
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from config.config_loader import ENV_CONFIG, get_config

console = Console(soft_wrap=True)

app = typer.Typer(help="Просмотр загруженной конфигурации", invoke_without_command=True)


@app.callback(invoke_without_command=True)
def _config_root(
    ctx: typer.Context,
    full: bool = typer.Option(False, "--full", "-f", help="Вывести весь объединённый YAML"),
) -> None:
    if ctx.invoked_subcommand is not None:
        return
    cfg = get_config()
    path = cfg.get_config_path()
    if full:
        data = cfg.get_merged_dict()
        dump = yaml.safe_dump(
            data,
            allow_unicode=True,
            default_flow_style=False,
            sort_keys=False,
        )
        title = f"Файл: {path}" if path else "Встроенные значения по умолчанию (файл не найден)"
        console.print(
            Panel(
                dump,
                title=f"[bold]{title}[/bold]",
                box=box.ROUNDED,
                border_style="bright_blue",
            )
        )
        return

    t = Table(
        title="[bold]nginx-lens — активный конфиг[/bold]",
        show_header=True,
        header_style="bold cyan",
        box=box.ROUNDED,
        border_style="dim",
        show_lines=True,
    )
    t.add_column("Параметр", style="cyan", no_wrap=True)
    t.add_column("Значение", overflow="fold", style="white")
    out = cfg.get_output_config()
    dft = cfg.get_defaults()
    vld = cfg.get_validate_config()
    lgc = cfg.get_logs_config()
    t.add_row("Файл конфигурации", path or "[dim](нет — используются встроенные defaults)[/dim]")
    t.add_row("output.colors", str(out.get("colors", True)))
    t.add_row("output.format", str(out.get("format", "table")))
    t.add_row("defaults.top (logs --top)", str(dft.get("top", 10)))
    t.add_row("logs.detect_anomalies", str(lgc.get("detect_anomalies", False)))
    t.add_row("validate.nginx_path (syntax, validate)", str(vld.get("nginx_path", "nginx")))
    t.add_row("defaults.nginx_config_path", str(dft.get("nginx_config_path", "")))
    t.add_row(ENV_CONFIG, os.environ.get(ENV_CONFIG, "[dim]не задан[/dim]"))
    console.print(t)
