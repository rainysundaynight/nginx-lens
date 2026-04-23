import os

# Цвета из конфига: до импорта Rich в командах
from config.config_loader import get_config

if not get_config().get_output_config().get("colors", True):
    os.environ["NO_COLOR"] = "1"

import typer
from rich.console import Console
from commands.health import health
from commands.analyze import analyze
from commands.tree import tree
from commands.diff import diff
from commands.route import route
from commands.include import include_tree
from commands.graph import graph
from commands.logs import logs
from commands.syntax import syntax
from commands.resolve import resolve
from commands.validate import validate
from commands.metrics import metrics
from commands.completion import app as completion_app
from commands.init import init
from commands.config_cmd import app as config_app

app = typer.Typer(help="nginx-lens — анализ и диагностика конфигураций Nginx")
console = Console()


@app.callback()
def _main(
    _ctx: typer.Context,
    version: bool = typer.Option(
        False,
        "--version",
        "-V",
        help="Показать версию пакета и выйти",
        is_flag=True,
        is_eager=True,
    ),
) -> None:
    """Опции верхнего уровня (перед именем подкоманды)."""
    if not version:
        return
    from utils.version import get_version
    import sys

    # stdout, без rich — удобно для скриптов и | grep
    print(f"nginx-lens {get_version()}", file=sys.stdout)
    raise typer.Exit(0)


app.command()(health)
app.command()(analyze)
app.command()(tree)
app.command()(diff)
app.command()(route)
app.command()(include_tree)
app.command()(graph)
app.command()(logs)
app.command()(syntax)
app.command()(resolve)
app.command()(validate)
app.command()(metrics)
app.command()(init)
app.add_typer(completion_app, name="completion", help="Генерация скриптов автодополнения")
app.add_typer(config_app, name="config", help="Показать загруженный конфиг (файл, ключевые поля, --full для YAML)")

if __name__ == "__main__":
    app() 