"""
Команда upstreams: сводка по named upstream, бэкендам, опциям и ссылкам на server/location.
"""

import json
import sys
from collections import defaultdict
from typing import Any, Dict, List, Optional

import typer
from rich import box
from rich.console import Console, Group
from rich.panel import Panel
from rich.table import Table
from rich.text import Text

from config.config_loader import get_config
from parser.nginx_parser import parse_nginx_config
from utils.upstream_inspect import (
    collect_known_upstream_names,
    find_upstream_references,
    iter_upstream_blocks,
    parse_server_options,
)

console = Console()


def _config_path_from_cli(config_path: Optional[str]) -> str:
    if config_path:
        return config_path
    cfg = get_config()
    p = cfg.get_nginx_config_path()
    if not p:
        console.print(
            "[red]Путь к nginx.conf не указан и не найден автоматически.[/red]"
        )
        console.print(
            "[yellow]Укажите путь через аргумент или настройте nginx_config_path в конфиге.[/yellow]"
        )
        sys.exit(1)
    return p


def _build_json(
    path: str,
    blocks: List[Dict[str, Any]],
    refs: List,
) -> Dict[str, Any]:
    ref_by: Dict[str, List[Dict]] = defaultdict(list)
    for r in refs:
        ref_by[r.upstream_name].append(
            {
                "server_name": r.server_name,
                "listen": r.listen,
                "location": r.location,
                "directive": r.from_directive,
                "value": r.value,
                "file": r.config_file,
                "stream": r.is_stream,
            }
        )
    out: List[Dict[str, Any]] = []
    for d in blocks:
        n = d.get("upstream") or "—"
        out.append(
            {
                "name": n,
                "defined_in": d.get("__file__") or "—",
                "servers": d.get("servers", []),
                "options": d.get("options", []),
                "referenced_by": ref_by.get(n, []),
            }
        )
    return {"config": path, "upstreams": out}


def upstreams(
    config_path: Optional[str] = typer.Argument(
        None,
        help="Путь к nginx.conf (если не указан — из конфига или автопоиск)",
    ),
    name: Optional[str] = typer.Option(
        None,
        "--name",
        "-n",
        help="Показать только upstream с этим именем",
    ),
    as_json: bool = typer.Option(
        False,
        "--json",
        help="Экспорт в JSON (stdout)",
    ),
) -> None:
    """
    Показать named upstream: бэкенды, директивы блока, где используется (server_name, listen, location) и файлы.
    """
    path = _config_path_from_cli(config_path)
    try:
        tree = parse_nginx_config(path)
    except FileNotFoundError:
        console.print(f"[red]Файл {path} не найден.[/red]")
        raise typer.Exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при разборе {path}: {e}[/red]")
        raise typer.Exit(1)

    all_blocks = list(iter_upstream_blocks(tree.directives))
    if not all_blocks:
        if as_json:
            print(
                json.dumps(
                    {"config": path, "upstreams": []},
                    ensure_ascii=False,
                    indent=2,
                )
            )
        else:
            console.print(
                "[yellow]В конфигурации не найдено ни одного блока upstream.[/yellow]"
            )
        raise typer.Exit(0)

    known = collect_known_upstream_names(tree.directives)
    refs = find_upstream_references(tree.directives, known)
    blocks = all_blocks
    if name:
        blocks = [b for b in blocks if b.get("upstream") == name]
        refs = [r for r in refs if r.upstream_name == name]
        if not blocks:
            console.print(
                f"[red]Блока upstream «{name}» в конфигурации нет.[/red]"
            )
            raise typer.Exit(1)

    names_sorted = sorted({b["upstream"] for b in all_blocks})
    if name:
        names_sorted = [name]

    if as_json:
        to_dump = all_blocks if not name else blocks
        jrefs = find_upstream_references(tree.directives, known)
        if name:
            jrefs = [r for r in jrefs if r.upstream_name == name]
        data = _build_json(path, to_dump, jrefs)
        print(json.dumps(data, ensure_ascii=False, indent=2))
        raise typer.Exit(0)

    for uname in names_sorted:
        udefs = [b for b in blocks if b.get("upstream") == uname]
        urefs = [r for r in refs if r.upstream_name == uname]
        for idx, udef in enumerate(udefs):
            _render_one_upstream(
                udef, urefs, uname, show_index=idx, total_same=len(udefs)
            )
        if not udefs and urefs:
            _render_missing_upstream(uname, urefs)

    n_total = len(names_sorted)
    console.print()
    t = Text()
    t.append("Найдено ", style="dim")
    t.append(f"{n_total} ", style="bold")
    t.append("имени upstream" if n_total == 1 else "имён upstream", style="dim")
    console.print(t)


def _render_missing_upstream(
    uname: str, urefs: List
) -> None:
    if not urefs:
        return
    title = f"[bold bright_cyan]{uname}[/] [dim]— нет разобранного upstream-блока (есть только ссылки)[/]"
    p = _refs_panel_only(uname, urefs, title)
    console.print(p)
    console.print()


def _refs_panel_only(uname: str, urefs: List, title: str) -> Panel:
    ref_table = _refs_table(urefs)
    return Panel(
        ref_table,
        title=Text.from_markup(title),
        box=box.ROUNDED,
        border_style="yellow",
    )


def _render_one_upstream(
    udef: Dict[str, Any],
    all_refs: List,
    uname: str,
    show_index: int,
    total_same: int,
) -> None:
    fpath = udef.get("__file__") or "—"
    urefs = [r for r in all_refs if r.upstream_name == uname]
    sub = ""
    if total_same > 1:
        sub = f" [dim](определение {show_index + 1}/{total_same})[/]"

    title = Text()
    title.append("upstream ", style="bold white")
    title.append(uname, style="bold bright_cyan")
    title.append(sub)

    subtitle = Text(f"файл: {fpath}", style="dim cyan")

    part_opts: List = []
    options = udef.get("options") or []
    if options:
        opt_tab = Table(
            show_header=True,
            header_style="bold green",
            box=box.SIMPLE,
            title="[green]Параметры группы[/]",
        )
        opt_tab.add_column("директива", style="green")
        opt_tab.add_column("значение", style="white")
        for o in options:
            opt_tab.add_row(
                o.get("directive", "—"),
                o.get("args", "") or "—",
            )
        part_opts.append(opt_tab)
    else:
        part_opts.append(
            Text("Параметры группы: [dim]— (только server)[/]", style="white")
        )

    srv = udef.get("servers", [])
    back = Table(
        show_header=True,
        header_style="bold magenta",
        box=box.SIMPLE_HEAVY,
        title="[magenta]Бэкенды (server ...)[/]",
    )
    back.add_column("#", style="dim", width=3, justify="right")
    back.add_column("Адрес", style="white")
    back.add_column("w", style="dim", max_width=5)
    back.add_column("max_f", style="dim", max_width=6)
    back.add_column("fail_t", style="dim", max_width=8)
    back.add_column("B", max_width=3)  # backup
    back.add_column("D", max_width=3)  # down
    back.add_column("другое", style="dim")
    for i, line in enumerate(srv, 1):
        p = parse_server_options(line)
        back.add_row(
            str(i),
            p["address"],
            p["weight"],
            p["max_fails"],
            p["fail_timeout"],
            p["backup"],
            p["down"],
            p["other"] if p["other"] != "—" else "—",
        )
    if not srv:
        back.add_row("—", "[dim]нет строк server[/]", "—", "—", "—", "—", "—", "—")

    ref_table = _refs_table(urefs) if urefs else Text(
        "[dim]Ни один server/location не ссылается на этот upstream по имени (proxy_pass, fastcgi_pass, …).[/]"
    )
    if urefs and isinstance(ref_table, Table):
        ref_table.title = "[yellow]Где используется[/]"

    body = Group(
        subtitle,
        Text(""),
        *part_opts,
        Text(""),
        back,
        Text(""),
        ref_table,
    )

    panel = Panel(
        body,
        title=title,
        box=box.ROUNDED,
        border_style="bright_blue",
        padding=(1, 2),
    )
    console.print(panel)
    console.print()


def _refs_table(urefs: List) -> Table:
    ref_table = Table(
        show_header=True,
        header_style="bold yellow",
        box=box.ROUNDED,
    )
    ref_table.add_column("server_name", style="blue", no_wrap=True)
    ref_table.add_column("listen", style="white", no_wrap=True, max_width=18)
    ref_table.add_column("location", style="magenta", max_width=24)
    ref_table.add_column("директива", style="green")
    ref_table.add_column("значение", style="white", max_width=40)
    ref_table.add_column("файл", style="dim", max_width=32)
    ref_table.add_column("стрим", max_width=5)
    for r in urefs:
        val = (r.value or "")[: 200] + ("…" if len(r.value or "") > 200 else "")
        ref_table.add_row(
            r.server_name,
            r.listen,
            r.location_display(),
            r.from_directive,
            val,
            r.config_file,
            "да" if r.is_stream else "—",
        )
    return ref_table
