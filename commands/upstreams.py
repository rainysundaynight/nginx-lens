"""
Команда upstreams: сводка по named upstream, бэкендам, опциям и ссылкам на server/location.
"""

import json
import os
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

_OPT_LINE_MAX = 100


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


def _shorten_path(path: str, max_len: int = 44) -> str:
    if not path or path == "—":
        return path
    s = str(path)
    if len(s) <= max_len:
        return s
    base = os.path.basename(s) or s
    if len(base) <= max_len and "/" in s:
        return f"…/{base}"
    return s[: max_len - 1] + "…"


def _options_oneline(options: List[Dict[str, str]]) -> str:
    if not options:
        return "—"
    parts: List[str] = []
    for o in options:
        d, a = o.get("directive", ""), (o.get("args") or "").strip()
        if a:
            parts.append(f"{d} {a}")
        else:
            parts.append(d)
    line = " · ".join(parts)
    if len(line) > _OPT_LINE_MAX:
        return line[: _OPT_LINE_MAX - 1] + "…"
    return line


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


def _render_table_compact(
    path: str,
    blocks: List[Dict[str, Any]],
    refs: List,
    names_sorted: List[str],
) -> None:
    console.print(
        f"[dim]Конфиг:[/] [white]{_shorten_path(path, 72)}[/]\n", highlight=False
    )

    # 1) Один блок = одна строка: сводка
    t1 = Table(
        show_header=True,
        header_style="bold cyan",
        box=box.SIMPLE,
        title="[bold]Блоки upstream[/]",
    )
    t1.add_column("upstream", style="bold bright_cyan", no_wrap=True)
    t1.add_column("файл", style="dim", max_width=40)
    t1.add_column("# srv", justify="right", style="magenta")
    t1.add_column("параметры группы", style="green", max_width=48)
    t1.add_column("# ссылок", justify="right", style="yellow")

    for b in blocks:
        un = b.get("upstream") or "—"
        fpath = b.get("__file__") or "—"
        sames = [x for x in blocks if x.get("upstream") == un]
        n_same = len(sames)
        if n_same > 1:
            k = sames.index(b) + 1
            label = f"{un} [dim]({k}/{n_same})[/]"
        else:
            label = un
        ref_n = len([r for r in refs if r.upstream_name == un])
        t1.add_row(
            label,
            _shorten_path(fpath, 40),
            str(len(b.get("servers", []))),
            _options_oneline(b.get("options") or []),
            str(ref_n),
        )
    console.print(t1)
    console.print()

    # 2) Все бэкенды: плоская таблица
    t2 = Table(
        show_header=True,
        header_style="bold magenta",
        box=box.SIMPLE,
        title="[bold]Серверы (внутри upstream)[/]",
    )
    t2.add_column("upstream", style="bright_cyan", no_wrap=True)
    t2.add_column("файл", style="dim", max_width=32)
    t2.add_column("#", justify="right", style="dim", width=3)
    t2.add_column("адрес", style="white")
    t2.add_column("w", style="dim", width=4)
    t2.add_column("max_f", style="dim", width=5)
    t2.add_column("B", width=2)
    t2.add_column("D", width=2)
    t2.add_column("другое", style="dim", max_width=20)

    for b in blocks:
        un = b.get("upstream") or "—"
        fpath = b.get("__file__") or "—"
        srvs = b.get("servers", [])
        if not srvs:
            t2.add_row(
                un,
                _shorten_path(fpath, 32),
                "—",
                "[dim]—[/]",
                "—",
                "—",
                "—",
                "—",
                "—",
            )
        else:
            for i, line in enumerate(srvs, 1):
                p = parse_server_options(line)
                t2.add_row(
                    un,
                    _shorten_path(fpath, 32),
                    str(i),
                    p["address"],
                    p["weight"],
                    p["max_fails"],
                    p["backup"] if p["backup"] != "—" else "—",
                    p["down"] if p["down"] != "—" else "—",
                    p["other"] if p["other"] != "—" else "—",
                )
    console.print(t2)
    console.print()

    # 3) Где используется
    t3 = _refs_table(
        refs,
        show_upstream=True,
        compact_path=True,
        compact_stream=True,
        box_style=box.SIMPLE,
    )
    t3.title = "[bold]Где используется[/]"
    if not refs:
        console.print("[dim]Нет ссылок (proxy_pass / fastcgi_pass / …) на named upstream.[/]\n")
    else:
        console.print(t3)
    console.print()

    n_names = len(names_sorted)
    n_def = len(blocks)
    n_ref = len(refs)
    console.print(
        f"[dim]Итого: {n_names} имён, {n_def} определений блока, {n_ref} ссылок[/]"
    )


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
    panels: bool = typer.Option(
        False,
        "--panels",
        help="Панель на каждый upstream (подробно; по умолчанию — компактные таблицы).",
    ),
) -> None:
    """
    Показать named upstream: бэкенды, директивы блока, где используется (server_name, listen, location) и файлы.

    По умолчанию — три компактные таблицы. Флаг [cyan]--panels[/] включает прежний подробный вид с панелями.
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

    if not panels:
        _render_table_compact(path, blocks, refs, names_sorted)
    else:
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
    ref_table = _refs_table(urefs, show_upstream=False, box_style=box.ROUNDED)
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
    back.add_column("B", max_width=3)
    back.add_column("D", max_width=3)
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

    ref_table = _refs_table(urefs, show_upstream=False) if urefs else Text(
        "[dim]Ни один server/location не ссылается на этот upstream по имени (proxy_pass, fastcgi_pass, …).[/]"
    )
    if urefs and isinstance(ref_table, Table):
        ref_table.title = "[yellow]Где используется[/]"

    body = Group(
        Text(f"файл: {fpath}", style="dim cyan"),
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


def _refs_table(
    urefs: List,
    show_upstream: bool = False,
    *,
    compact_path: bool = False,
    compact_stream: bool = False,
    box_style=box.ROUNDED,
) -> Table:
    ref_table = Table(
        show_header=True,
        header_style="bold yellow",
        box=box_style,
    )
    if show_upstream:
        ref_table.add_column("upstream", style="bright_cyan", no_wrap=True)
    ref_table.add_column("server_name", style="blue", no_wrap=True, max_width=20)
    ref_table.add_column("listen", style="white", no_wrap=True, max_width=16)
    ref_table.add_column("location", style="magenta", max_width=20)
    ref_table.add_column("директива", style="green", max_width=14)
    ref_table.add_column("значение", style="white", max_width=36)
    ref_table.add_column("файл", style="dim", max_width=28)
    ref_table.add_column("стрим", width=5)
    for r in urefs:
        val = (r.value or "")
        if len(val) > 100:
            val = val[:99] + "…"
        path_cell = _shorten_path(r.config_file, 28) if compact_path else (r.config_file or "—")
        if compact_stream:
            sm = "S" if r.is_stream else "·"
        else:
            sm = "да" if r.is_stream else "—"
        row = [
            r.server_name,
            r.listen,
            r.location_display(),
            r.from_directive,
            val,
            path_cell,
            sm,
        ]
        if show_upstream:
            row.insert(0, r.upstream_name)
        ref_table.add_row(*row)
    return ref_table
