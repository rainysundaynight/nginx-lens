import os
import sys

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

app = typer.Typer(
    help="nginx-lens — анализ и диагностика конфигураций Nginx",
    # иначе Typer требует подкоманду и не принимает только глобальные флаги на группе
    invoke_without_command=True,
    context_settings={"help_option_names": ["-h", "--help"]},
)
console = Console()


def main() -> None:
    """
    Точка входа консольного скрипта. Обрабатываем --version/-V до Typer, иначе Click
    оставляет 'Missing command' у группы с subcommands.
    """
    if len(sys.argv) > 1 and sys.argv[1] in ("--version", "-V"):
        from utils.version import get_version

        print(f"nginx-lens {get_version()}", file=sys.stdout)
        raise SystemExit(0)
    if len(sys.argv) == 1:
        sys.argv.append("--help")
    app()


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
    main()