import sys
from typing import Optional
import typer
from rich.console import Console
from rich.table import Table
from analyzer.conflicts import find_location_conflicts, find_listen_servername_conflicts
from analyzer.duplicates import find_duplicate_directives
from analyzer.empty_blocks import find_empty_blocks
from analyzer.warnings import find_warnings
from analyzer.unused import find_unused_variables
from parser.nginx_parser import parse_nginx_config
from analyzer.rewrite import find_rewrite_issues
from analyzer.dead_locations import find_dead_locations
from exporter.json_yaml import format_analyze_results, print_export
from config.config_loader import get_config

app = typer.Typer()
console = Console()

# Карта советов и критичности для issue_type
ISSUE_META = {
    'location_conflict': ("Возможное пересечение location. Это не всегда ошибка: порядок и типы location могут быть корректны. Проверьте, что порядок и типы location соответствуют вашим ожиданиям. Если всё ок — игнорируйте предупреждение.", "medium"),
    'duplicate_directive': ("Оставьте только одну директиву с нужным значением в этом блоке.", "medium"),
    'empty_block': ("Удалите или заполните пустой блок.", "low"),
    'proxy_pass_no_scheme': ("Добавьте http:// или https:// в proxy_pass.", "medium"),
    'autoindex_on': ("Отключите autoindex, если не требуется публикация файлов.", "medium"),
    'if_block': ("Избегайте if внутри location, используйте map/try_files.", "medium"),
    'server_tokens_on': ("Отключите server_tokens для безопасности.", "low"),
    'ssl_missing': ("Укажите путь к SSL-сертификату/ключу.", "high"),
    'ssl_protocols_weak': ("Отключите устаревшие протоколы TLS.", "high"),
    'ssl_ciphers_weak': ("Используйте современные шифры.", "high"),
    'listen_443_no_ssl': ("Добавьте ssl к listen 443.", "high"),
    'listen_443_no_http2': ("Добавьте http2 к listen 443 для производительности.", "low"),
    'no_limit_req_conn': ("Добавьте limit_req/limit_conn для защиты от DDoS.", "medium"),
    'missing_security_header': ("Добавьте security-заголовок.", "medium"),
    'deprecated': ("Замените устаревшую директиву.", "medium"),
    'limit_too_small': ("Увеличьте лимит до рекомендуемого значения.", "medium"),
    'limit_too_large': ("Уменьшите лимит до разумного значения.", "medium"),
    'unused_variable': ("Удалите неиспользуемую переменную.", "low"),
    'listen_servername_conflict': ("Измените listen/server_name для устранения конфликта.", "high"),
    'rewrite_cycle': ("Проверьте rewrite на циклические правила.", "high"),
    'rewrite_conflict': ("Проверьте порядок и уникальность rewrite.", "medium"),
    'rewrite_no_flag': ("Добавьте last/break/redirect/permanent к rewrite.", "low"),
    'dead_location': ("Удалите неиспользуемый location или используйте его.", "low"),
}
SEVERITY_COLOR = {"high": "red", "medium": "orange3", "low": "yellow"}

def analyze(
    config_path: Optional[str] = typer.Argument(None, help="Путь к nginx.conf (если не указан, используется из конфига или автопоиск)"),
    json: bool = typer.Option(False, "--json", help="Экспортировать результаты в JSON"),
    yaml: bool = typer.Option(False, "--yaml", help="Экспортировать результаты в YAML"),
):
    """
    Анализирует конфигурацию Nginx на типовые проблемы и best practices.

    Показывает:
      - Конфликты location-ов
      - Дублирующиеся директивы
      - Пустые блоки
      - Потенциальные проблемы (proxy_pass без схемы, autoindex on, if, server_tokens on, SSL, лимиты, deprecated и др.)
      - Неиспользуемые переменные
      - Конфликты listen/server_name
      - Проблемы с rewrite
      - Мертвые location-ы

    Пример:
        nginx-lens analyze /etc/nginx/nginx.conf
        nginx-lens analyze  # Использует путь из конфига
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
    conflicts = find_location_conflicts(tree)
    dups = find_duplicate_directives(tree)
    empties = find_empty_blocks(tree)
    warnings = find_warnings(tree)
    unused_vars = find_unused_variables(tree)
    listen_conflicts = find_listen_servername_conflicts(tree)
    rewrite_issues = find_rewrite_issues(tree)
    dead_locations = find_dead_locations(tree)

    table = Table(show_header=True, header_style="bold blue")
    table.add_column("issue_type")
    table.add_column("issue_description")
    table.add_column("solution")

    def add_row(issue_type, desc):
        solution, severity = ISSUE_META.get(issue_type, ("", "low"))
        color = SEVERITY_COLOR.get(severity, "yellow")
        table.add_row(f"[{color}]{issue_type}[/{color}]", desc, f"[{color}]{solution}[/{color}]")

    for c in conflicts:
        add_row("location_conflict", f"server: {c['server'].get('arg', '')} location: {c['location1']} ↔ {c['location2']}")
    for d in dups:
        loc = d.get('location')
        add_row("duplicate_directive", f"{d['directive']} ({d['args']}) — {d['count']} раз в блоке {d['block'].get('block', d['block'])}{' location: '+str(loc) if loc else ''}")
    for e in empties:
        add_row("empty_block", f"{e['block']} {e['arg'] or ''}")
    for w in warnings:
        t = w['type']
        if t == 'proxy_pass_no_scheme':
            add_row(t, f"proxy_pass без схемы: {w['value']}")
        elif t == 'autoindex_on':
            add_row(t, f"autoindex on в блоке {w['context'].get('block','')}")
        elif t == 'if_block':
            add_row(t, f"директива if внутри блока {w['context'].get('block','')}")
        elif t == 'server_tokens_on':
            add_row(t, f"server_tokens on в блоке {w['context'].get('block','')}")
        elif t == 'ssl_missing':
            add_row(t, f"{w['directive']} не указан")
        elif t == 'ssl_protocols_weak':
            add_row(t, f"ssl_protocols содержит устаревшие протоколы: {w['value']}")
        elif t == 'ssl_ciphers_weak':
            add_row(t, f"ssl_ciphers содержит слабые шифры: {w['value']}")
        elif t == 'listen_443_no_ssl':
            add_row(t, f"listen без ssl: {w['value']}")
        elif t == 'listen_443_no_http2':
            add_row(t, f"listen 443 без http2: {w['value']}")
        elif t == 'no_limit_req_conn':
            add_row(t, f"server без limit_req/limit_conn")
        elif t == 'missing_security_header':
            add_row(t, f"отсутствует security header: {w['value']}")
        elif t == 'deprecated':
            add_row(t, f"устаревшая директива: {w['directive']} — {w['value']}")
        elif t == 'limit_too_small':
            add_row(t, f"слишком маленькое значение: {w['directive']} = {w['value']}")
        elif t == 'limit_too_large':
            add_row(t, f"слишком большое значение: {w['directive']} = {w['value']}")
    for v in unused_vars:
        add_row("unused_variable", v['name'])
    for c in listen_conflicts:
        add_row("listen_servername_conflict", f"server1: {c['server1'].get('arg','')} server2: {c['server2'].get('arg','')} listen: {','.join(c['listen'])} server_name: {','.join(c['server_name'])}")
    for r in rewrite_issues:
        add_row(r['type'], r['value'])
    for l in dead_locations:
        add_row("dead_location", f"server: {l['server'].get('arg','')} location: {l['location'].get('arg','')}")

    if table.row_count == 0:
        console.print("[green]Проблем не найдено[/green]")
    else:
        console.print(table) 