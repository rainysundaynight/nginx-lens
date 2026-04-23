import os
import sys
from typing import Optional, List, Dict, Any
import typer
import gzip
from datetime import datetime, timedelta
from collections import Counter, defaultdict

from rich import box
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from exporter.json_yaml import format_logs_results, print_export
from exporter.csv import export_logs_to_csv
from config.config_loader import get_config
from utils.log_parsers import parse_universal_log_line

_LOGS_DEFAULT_DETECT = get_config().get_logs_config().get("detect_anomalies", False)

app = typer.Typer(
    help="Анализ access-логов: nginx, JSON, LTSV и нестрогий текст — подгоняется к файлу сам, без настроек.",
)
console = Console()

# --- оформление вывода ---------------------------------------------------------

_RT_LABELS = {
    "min": "Минимум",
    "max": "Максимум",
    "avg": "Среднее",
    "median": "Медиана",
    "p95": "p95",
    "p99": "p99",
    "total_requests_with_time": "Запросов с таймингом",
}

_ANOMALY_TYPE_RU = {
    "error_spike": "Всплеск ошибок",
    "slow_requests": "Медленные ответы",
    "suspicious_ips": "Активные IP",
    "unusual_paths": "Частые пути",
}

_SEVER_RU = {"high": "высокая", "medium": "средняя", "low": "низкая"}


def _http_status_badge(status: str) -> str:
    if not (status and status.isdigit() and len(status) == 3):
        return f"[dim]{status}[/dim]"
    c = {2: "green", 3: "cyan", 4: "yellow", 5: "red"}.get(int(status[0]), "white")
    return f"[bold {c}]{status}[/bold {c}]"


def _ellipsize(s: str, max_len: int = 96) -> str:
    s = s.replace("\n", " ")
    if len(s) <= max_len:
        return s
    return s[: max_len - 1] + "…"


def _short_path_display(path: str, max_len: int = 88) -> str:
    if len(path) <= max_len:
        return path
    a = max_len // 2 - 2
    b = max_len - a - 1
    return f"{path[:a]}…{path[-b:]}"


def _build_summary_panel(
    log_path: str,
    n: int,
    err_n: int,
    since: Optional[str],
    until: Optional[str],
    status: Optional[str],
) -> Panel:
    abs_path = os.path.abspath(os.path.expanduser(log_path))
    path_show = _ellipsize(abs_path, 90)
    err_pct = (100.0 * err_n / n) if n else 0.0
    line1 = f"[bold]Файл[/bold]  [dim]{path_show}[/dim]"
    if log_path.endswith(".gz"):
        line1 += "  [dim](gzip)[/dim]"
    line2 = (
        f"Запросов: [green bold]{n:,}[/green bold]  [dim]·[/dim]  "
        f"4xx/5xx: [yellow bold]{err_n:,}[/yellow bold]  [dim]({err_pct:.1f} %)[/dim]"
    )
    rest: list[str] = []
    if since or until or status:
        bits = []
        if since:
            bits.append(f"с {since}")
        if until:
            bits.append(f"до {until}")
        if status:
            bits.append(f"статусы: {status}")
        rest.append(f"[dim]Фильтр:[/dim]  {', '.join(bits)}")
    content = "\n".join([line1, line2] + rest)
    return Panel(
        content,
        title="[bold]nginx-lens — сводка по логу[/bold]",
        title_align="left",
        box=box.ROUNDED,
        border_style="bright_blue",
        padding=(1, 2),
    )


def _data_table_title(text: str) -> str:
    return f"[bold white]{text}[/bold white]"


def _new_table(
    title: str, *, with_lines: bool = True, border: str = "bright_black"
) -> Table:
    return Table(
        title=_data_table_title(title),
        show_header=True,
        header_style="bold cyan",
        box=box.ROUNDED,
        border_style=border,
        show_lines=with_lines,
        padding=(0, 1),
        title_justify="left",
    )


def logs(
    log_path: str = typer.Argument(..., help="Путь к access.log или error.log"),
    top: Optional[int] = typer.Option(None, help="Сколько топ-значений выводить"),
    json: bool = typer.Option(False, "--json", help="Экспортировать результаты в JSON"),
    yaml: bool = typer.Option(False, "--yaml", help="Экспортировать результаты в YAML"),
    csv: bool = typer.Option(False, "--csv", help="Экспортировать результаты в CSV"),
    since: Optional[str] = typer.Option(None, "--since", help="Фильтр: с даты (формат: YYYY-MM-DD или YYYY-MM-DD HH:MM:SS)"),
    until: Optional[str] = typer.Option(None, "--until", help="Фильтр: до даты (формат: YYYY-MM-DD или YYYY-MM-DD HH:MM:SS)"),
    status: Optional[str] = typer.Option(None, "--status", help="Фильтр по статусам (например: 404,500)"),
    detect_anomalies: bool = typer.Option(
        _LOGS_DEFAULT_DETECT,
        "--detect-anomalies/--no-detect-anomalies",
        help="Поиск аномалий (по умолчанию: из конфига [logs.detect_anomalies])",
    ),
):
    """
    Анализирует access.log/error.log.

    Показывает:
      - Топ HTTP-статусов (404, 500 и др.)
      - Топ путей
      - Топ IP-адресов
      - Топ User-Agent
      - Топ путей с ошибками 404/500
      - Анализ времени ответа (если доступно)
      - Обнаружение аномалий

    Пример:
        nginx-lens logs /var/log/nginx/access.log --top 20
        nginx-lens logs /path/to/anything.log
    """
    # Загружаем конфигурацию
    config = get_config()
    defaults = config.get_defaults()
    
    # Применяем значения из конфига, если не указаны через CLI
    top = top if top is not None else defaults.get("top", 10)
    
    # Парсинг фильтров
    status_filter = None
    if status:
        status_filter = set(s.strip() for s in status.split(','))
    
    since_dt = None
    if since:
        try:
            if len(since) == 10:  # YYYY-MM-DD
                since_dt = datetime.strptime(since, "%Y-%m-%d")
            else:  # YYYY-MM-DD HH:MM:SS
                since_dt = datetime.strptime(since, "%Y-%m-%d %H:%M:%S")
        except ValueError:
            console.print(f"[red]Неверный формат даты для --since: {since}. Используйте YYYY-MM-DD или YYYY-MM-DD HH:MM:SS[/red]")
            sys.exit(1)
    
    until_dt = None
    if until:
        try:
            if len(until) == 10:  # YYYY-MM-DD
                until_dt = datetime.strptime(until, "%Y-%m-%d") + timedelta(days=1)
            else:  # YYYY-MM-DD HH:MM:SS
                until_dt = datetime.strptime(until, "%Y-%m-%d %H:%M:%S")
        except ValueError:
            console.print(f"[red]Неверный формат даты для --until: {until}. Используйте YYYY-MM-DD или YYYY-MM-DD HH:MM:SS[/red]")
            sys.exit(1)
    
    # Чтение лога построчно (поддержка gzip, без загрузки всего файла в память)
    try:
        if log_path.endswith(".gz"):
            _log = gzip.open(log_path, "rt", encoding="utf-8", errors="ignore")
        else:
            _log = open(log_path, "r", encoding="utf-8", errors="ignore")
    except FileNotFoundError:
        console.print(f"[red]Файл {log_path} не найден. Проверьте путь к логу.[/red]")
        sys.exit(1)
    except OSError as e:
        console.print(f"[red]Ошибка при чтении {log_path}: {e}[/red]")
        sys.exit(1)
    status_counter = Counter()
    path_counter = Counter()
    ip_counter = Counter()
    user_agent_counter = Counter()
    errors = defaultdict(list)
    response_times = []
    log_entries = []

    with _log:
        for line in _log:
            pl = parse_universal_log_line(line)
            if pl is None:
                continue
            log_time = pl.time
            if (since_dt or until_dt) and log_time is None:
                continue
            try:
                if since_dt and log_time is not None and log_time < since_dt:
                    continue
                if until_dt and log_time is not None and log_time > until_dt:
                    continue
            except TypeError:
                continue

            status = pl.status
            path = pl.path
            ip = pl.ip
            method = pl.method
            user_agent = pl.user_agent
            if status_filter and status not in status_filter:
                continue

            entry = {
                "time": log_time,
                "ip": ip,
                "path": path,
                "status": status,
                "method": method,
                "user_agent": user_agent,
                "response_time": pl.response_time,
            }
            log_entries.append(entry)

            status_counter[status] += 1
            path_counter[path] += 1
            ip_counter[ip] += 1
            if user_agent:
                user_agent_counter[user_agent] += 1
            if status.startswith("4") or status.startswith("5"):
                errors[status].append(path)
            if pl.response_time is not None:
                response_times.append(pl.response_time)
    
    # Проверка на пустые результаты
    if not log_entries:
        filters_on = bool(since_dt or until_dt or status_filter)
        empty_msg = (
            "Нет записей после применения фильтров (дата и/или статусы). "
            "Строки без распознаваемой даты пропускаются при --since/--until."
            if filters_on
            else (
                "Не удалось выделить HTTP-запросы из файла. Нужны строки с путём и кодом ответа "
                "(как в access.log), иначе это может быть error.log, аудит в другом формате или пустой файл."
            )
        )
        if json or yaml or csv:
            empty_data = {
                "timestamp": __import__('datetime').datetime.now().isoformat(),
                "summary": {"total_requests": 0},
                "message": empty_msg,
            }
            if csv:
                print("Category,Type,Value,Count\nNo Data,,,,0")
            else:
                format_type = 'json' if json else 'yaml'
                print_export(empty_data, format_type)
        else:
            console.print(
                Panel(
                    f"[yellow]{empty_msg}[/yellow]",
                    title="[bold]Нет данных[/bold]",
                    box=box.ROUNDED,
                    border_style="yellow",
                    title_align="left",
                    padding=(1, 2),
                )
            )
        return
    
    # Анализ времени ответа
    response_time_stats = {}
    if response_times:
        response_time_stats = {
            "min": min(response_times),
            "max": max(response_times),
            "avg": sum(response_times) / len(response_times),
            "median": sorted(response_times)[len(response_times) // 2],
            "p95": sorted(response_times)[int(len(response_times) * 0.95)] if response_times else 0,
            "p99": sorted(response_times)[int(len(response_times) * 0.99)] if response_times else 0,
            "total_requests_with_time": len(response_times)
        }
    
    # Обнаружение аномалий
    anomalies = []
    if detect_anomalies:
        # Аномалия 1: Резкий скачок ошибок
        if len(log_entries) > 100:
            # Разбиваем на временные окна
            window_size = max(100, len(log_entries) // 10)
            error_rates = []
            for i in range(0, len(log_entries), window_size):
                window = log_entries[i:i+window_size]
                error_count = sum(1 for e in window if e['status'].startswith('4') or e['status'].startswith('5'))
                error_rates.append(error_count / len(window) if window else 0)
            
            if len(error_rates) > 1:
                avg_rate = sum(error_rates) / len(error_rates)
                for i, rate in enumerate(error_rates):
                    if rate > avg_rate * 2:  # Удвоение ошибок
                        anomalies.append({
                            "type": "error_spike",
                            "description": f"Резкий скачок ошибок в окне {i+1}: {rate*100:.1f}% (среднее: {avg_rate*100:.1f}%)",
                            "severity": "high"
                        })
        
        # Аномалия 2: Медленные запросы
        if response_times:
            slow_threshold = response_time_stats.get("p95", 1.0) * 2
            slow_requests = [e for e in log_entries if e.get('response_time') and e['response_time'] > slow_threshold]
            if slow_requests:
                anomalies.append({
                    "type": "slow_requests",
                    "description": f"Найдено {len(slow_requests)} медленных запросов (> {slow_threshold:.2f}s)",
                    "severity": "medium"
                })
        
        # Аномалия 3: Необычные паттерны IP
        if len(log_entries) > 50:
            ip_counts = Counter(e['ip'] for e in log_entries)
            avg_ip_requests = len(log_entries) / len(ip_counts) if ip_counts else 0
            suspicious_ips = [ip for ip, count in ip_counts.items() if count > avg_ip_requests * 5]
            if suspicious_ips:
                anomalies.append({
                    "type": "suspicious_ips",
                    "description": f"Подозрительная активность с IP: {', '.join(suspicious_ips[:5])}",
                    "severity": "medium"
                })
        
        # Аномалия 4: Необычные пути
        if len(log_entries) > 50:
            path_counts = Counter(e['path'] for e in log_entries)
            avg_path_requests = len(log_entries) / len(path_counts) if path_counts else 0
            unusual_paths = [path for path, count in path_counts.items() if count > avg_path_requests * 10]
            if unusual_paths:
                anomalies.append({
                    "type": "unusual_paths",
                    "description": f"Необычно много запросов к путям: {', '.join(unusual_paths[:3])}",
                    "severity": "low"
                })
    
    # Экспорт в CSV
    if csv:
        csv_output = export_logs_to_csv(
            status_counter, path_counter, ip_counter, user_agent_counter,
            errors, response_time_stats, anomalies
        )
        print(csv_output)
        return
    
    # Экспорт в JSON/YAML
    if json or yaml:
        export_data = format_logs_results(
            status_counter, path_counter, ip_counter, user_agent_counter, errors, top,
            response_time_stats if response_time_stats else None,
            anomalies if anomalies else None
        )
        format_type = 'json' if json else 'yaml'
        print_export(export_data, format_type)
        return

    n = len(log_entries)
    err_n = sum(1 for e in log_entries if e["status"].startswith("4") or e["status"].startswith("5"))
    parts: list = [
        _build_summary_panel(
            log_path, n, err_n, since, until, status,
        )
    ]
    if response_time_stats:
        t_rt = _new_table("Время ответа (сервер/прокси)", with_lines=True)
        t_rt.add_column("Метрика", style="dim", no_wrap=True)
        t_rt.add_column("Значение", justify="right", style="green")
        for metric, value in response_time_stats.items():
            label = _RT_LABELS.get(metric, metric.replace("_", " ").title())
            if metric == "total_requests_with_time":
                t_rt.add_row(label, f"[white]{int(value):,}[/white]")
            else:
                t_rt.add_row(label, f"{value:.3f} [dim]s[/dim]")
        parts.append(t_rt)
    if anomalies:
        t_an = _new_table("Аномалии", with_lines=True, border="red")
        t_an.add_column("Тип", width=20, no_wrap=True)
        t_an.add_column("Описание", ratio=1)
        t_an.add_column("Серьёзность", justify="right", width=12, no_wrap=True)
        for a in anomalies:
            typ = a.get("type", "")
            typ_d = _ANOMALY_TYPE_RU.get(typ, typ)
            sev = a.get("severity", "low")
            sc = {"high": "red", "medium": "orange1", "low": "yellow"}.get(sev, "white")
            sev_ru = _SEVER_RU.get(sev, sev)
            t_an.add_row(
                typ_d,
                a.get("description", ""),
                f"[{sc}]{sev_ru}[/]",
            )
        parts.append(t_an)
    t_st = _new_table("Статусы HTTP")
    t_st.add_column("Код", style="white", no_wrap=True)
    t_st.add_column("Счётчик", justify="right", style="bold")
    t_st.add_column("Доля", justify="right", style="dim")
    for st, count in status_counter.most_common(top):
        p = 100.0 * count / n
        t_st.add_row(_http_status_badge(st), f"{count:,}", f"{p:.1f} %")
    parts.append(t_st)
    t_p = _new_table("Самые частые пути", border="magenta")
    t_p.add_column("Путь", style="magenta", overflow="fold")
    t_p.add_column("Счётчик", justify="right", style="bold")
    t_p.add_column("Доля", justify="right", style="dim")
    for pth, count in path_counter.most_common(top):
        p = 100.0 * count / n
        t_p.add_row(_short_path_display(pth, 100), f"{count:,}", f"{p:.1f} %")
    parts.append(t_p)
    t_ip = _new_table("IP-адреса", border="bright_blue")
    t_ip.add_column("IP", style="bright_blue", no_wrap=True, overflow="ellipsis")
    t_ip.add_column("Счётчик", justify="right", style="bold")
    t_ip.add_column("Доля", justify="right", style="dim")
    for ip, count in ip_counter.most_common(top):
        p = 100.0 * count / n
        t_ip.add_row(ip, f"{count:,}", f"{p:.1f} %")
    parts.append(t_ip)
    if user_agent_counter:
        t_ua = _new_table("User-Agent", border="white")
        t_ua.add_column("Клиент", style="white", overflow="fold")
        t_ua.add_column("Счётчик", justify="right", style="bold")
        t_ua.add_column("Доля", justify="right", style="dim")
        for ua, count in user_agent_counter.most_common(top):
            p = 100.0 * count / n
            t_ua.add_row(_ellipsize(ua, 100), f"{count:,}", f"{p:.1f} %")
        parts.append(t_ua)
    for err in ("404", "500"):
        if not errors[err]:
            continue
        c = Counter(errors[err])
        t_e = _new_table(f"Пути с {err} — топ", border="yellow")
        t_e.add_column("Путь", style="yellow", overflow="fold")
        t_e.add_column("Счётчик", justify="right", style="bold")
        t_e.add_column("Доля", style="dim", justify="right")
        err_total = len(errors[err])
        for pth, count in c.most_common(top):
            p = 100.0 * count / err_total if err_total else 0
            t_e.add_row(_short_path_display(pth, 100), f"{count:,}", f"{p:.1f} %")
        parts.append(t_e)
    for i, block in enumerate(parts):
        if i:
            console.print()
        console.print(block)