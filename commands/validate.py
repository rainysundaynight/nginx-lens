import sys
import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich import box
import subprocess
import os
from typing import Optional, Tuple, List

from parser.nginx_parser import parse_nginx_config
from analyzer.conflicts import find_location_conflicts, find_listen_servername_conflicts
from analyzer.duplicates import find_duplicate_directives
from analyzer.empty_blocks import find_empty_blocks
from analyzer.warnings import find_warnings
from analyzer.unused import find_unused_variables
from analyzer.rewrite import find_rewrite_issues
from analyzer.dead_locations import find_dead_locations
from upstream_checker.checker import check_upstreams, resolve_upstreams
from upstream_checker.dns_cache import disable_cache, enable_cache
from config.config_loader import get_config

app = typer.Typer()
console = Console()

# Карта критичности для issue_type (из analyze.py)
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


def validate(
    config_path: str = typer.Argument(..., help="Путь к nginx.conf"),
    nginx_path: str = typer.Option("nginx", "--nginx-path", help="Путь к бинарю nginx (по умолчанию 'nginx')"),
    check_syntax: bool = typer.Option(True, "--syntax/--no-syntax", help="Проверять синтаксис через nginx -t"),
    check_analysis: bool = typer.Option(True, "--analysis/--no-analysis", help="Выполнять анализ проблем"),
    check_upstream: bool = typer.Option(True, "--upstream/--no-upstream", help="Проверять доступность upstream"),
    check_dns: bool = typer.Option(False, "--dns/--no-dns", help="Проверять DNS резолвинг upstream"),
    timeout: Optional[float] = typer.Option(None, "--timeout", help="Таймаут проверки upstream (сек)"),
    max_workers: Optional[int] = typer.Option(None, "--max-workers", "-w", help="Максимальное количество потоков для параллельной обработки"),
    json: bool = typer.Option(False, "--json", help="Экспортировать результаты в JSON"),
    yaml: bool = typer.Option(False, "--yaml", help="Экспортировать результаты в YAML"),
):
    """
    Комплексная валидация конфигурации Nginx.
    
    Выполняет:
      - Проверку синтаксиса (nginx -t)
      - Анализ проблем и best practices
      - Проверку доступности upstream серверов
      - Опционально: DNS резолвинг upstream
    
    Возвращает exit code 0 при успехе, 1 при наличии проблем.
    Подходит для использования в CI/CD пайплайнах.

    Пример:
        nginx-lens validate /etc/nginx/nginx.conf
        nginx-lens validate /etc/nginx/nginx.conf --no-upstream
        nginx-lens validate /etc/nginx/nginx.conf --dns --json
    """
    exit_code = 0
    results = {
        "syntax": {"valid": False, "errors": []},
        "analysis": {"issues": [], "summary": {}},
        "upstream": {"healthy": True, "servers": []},
        "dns": {"resolved": True, "servers": []}
    }
    
    # Загружаем конфигурацию
    config = get_config()
    defaults = config.get_defaults()
    cache_config = config.get_cache_config()
    validate_config = config.get_validate_config()
    
    # Применяем значения из конфига, если не указаны через CLI
    timeout_val = timeout if timeout is not None else defaults.get("timeout", 2.0)
    max_workers_val = max_workers if max_workers is not None else defaults.get("max_workers", 10)
    cache_ttl = cache_config.get("ttl", defaults.get("dns_cache_ttl", 300))
    nginx_path_val = nginx_path if nginx_path is not None else validate_config.get("nginx_path", "nginx")
    
    # Применяем настройки проверок из конфига, если не указаны через CLI
    check_syntax_val = check_syntax if check_syntax is not None else validate_config.get("check_syntax", True)
    check_analysis_val = check_analysis if check_analysis is not None else validate_config.get("check_analysis", True)
    check_upstream_val = check_upstream if check_upstream is not None else validate_config.get("check_upstream", True)
    check_dns_val = check_dns if check_dns is not None else validate_config.get("check_dns", False)
    
    # Управление кэшем
    enable_cache()
    
    # 1. Проверка синтаксиса
    if check_syntax_val:
        console.print(Panel("[bold blue]1. Проверка синтаксиса[/bold blue]", box=box.ROUNDED))
        syntax_valid, syntax_errors = _check_syntax(config_path, nginx_path_val)
        results["syntax"]["valid"] = syntax_valid
        results["syntax"]["errors"] = syntax_errors
        
        if syntax_valid:
            console.print("[green]✓ Синтаксис корректен[/green]")
        else:
            console.print("[red]✗ Обнаружены ошибки синтаксиса[/red]")
            for error in syntax_errors:
                console.print(f"  [red]{error}[/red]")
            exit_code = 1
    
    # 2. Анализ проблем
    tree = None
    if check_analysis_val:
        console.print(Panel("[bold blue]2. Анализ проблем и best practices[/bold blue]", box=box.ROUNDED))
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
        
        # Собираем все проблемы
        all_issues = []
        high_severity_count = 0
        
        for c in conflicts:
            advice, severity = ISSUE_META.get('location_conflict', ("", "medium"))
            all_issues.append({
                "type": "location_conflict",
                "severity": severity,
                "message": f"server: {c['server'].get('arg', '')} location: {c['location1']} ↔ {c['location2']}",
                "advice": advice
            })
            if severity == "high":
                high_severity_count += 1
        
        for d in dups:
            advice, severity = ISSUE_META.get('duplicate_directive', ("", "medium"))
            loc = d.get('location')
            all_issues.append({
                "type": "duplicate_directive",
                "severity": severity,
                "message": f"{d['directive']} ({d['args']}) — {d['count']} раз в блоке {d['block'].get('block', d['block'])}{' location: '+str(loc) if loc else ''}",
                "advice": advice
            })
            if severity == "high":
                high_severity_count += 1
        
        for e in empties:
            advice, severity = ISSUE_META.get('empty_block', ("", "low"))
            all_issues.append({
                "type": "empty_block",
                "severity": severity,
                "message": f"{e['block']} {e['arg'] or ''}",
                "advice": advice
            })
        
        for w in warnings:
            issue_type = w.get('type', '')
            advice, severity = ISSUE_META.get(issue_type, ("", "medium"))
            all_issues.append({
                "type": issue_type,
                "severity": severity,
                "message": w.get('value', ''),
                "advice": advice
            })
            if severity == "high":
                high_severity_count += 1
        
        for v in unused_vars:
            advice, severity = ISSUE_META.get('unused_variable', ("", "low"))
            all_issues.append({
                "type": "unused_variable",
                "severity": severity,
                "message": v.get('name', ''),
                "advice": advice
            })
        
        for c in listen_conflicts:
            advice, severity = ISSUE_META.get('listen_servername_conflict', ("", "high"))
            all_issues.append({
                "type": "listen_servername_conflict",
                "severity": severity,
                "message": f"server1: {c.get('server1', {}).get('arg','')} server2: {c.get('server2', {}).get('arg','')}",
                "advice": advice
            })
            high_severity_count += 1
        
        for r in rewrite_issues:
            advice, severity = ISSUE_META.get(r.get('type', ''), ("", "medium"))
            all_issues.append({
                "type": r.get('type', ''),
                "severity": severity,
                "message": r.get('value', ''),
                "advice": advice
            })
            if severity == "high":
                high_severity_count += 1
        
        for l in dead_locations:
            advice, severity = ISSUE_META.get('dead_location', ("", "low"))
            all_issues.append({
                "type": "dead_location",
                "severity": severity,
                "message": f"server: {l.get('server', {}).get('arg','')} location: {l.get('location', {}).get('arg','')}",
                "advice": advice
            })
        
        results["analysis"]["issues"] = all_issues
        results["analysis"]["summary"] = {
            "total": len(all_issues),
            "high": high_severity_count,
            "medium": sum(1 for i in all_issues if i["severity"] == "medium"),
            "low": sum(1 for i in all_issues if i["severity"] == "low")
        }
        
        if all_issues:
            # Показываем только high и medium проблемы в таблице
            table = Table(show_header=True, header_style="bold blue")
            table.add_column("Severity", style="bold")
            table.add_column("Type")
            table.add_column("Issue")
            
            for issue in all_issues:
                if issue["severity"] in ["high", "medium"]:
                    color = SEVERITY_COLOR.get(issue["severity"], "white")
                    table.add_row(
                        f"[{color}]{issue['severity'].upper()}[/{color}]",
                        issue["type"],
                        issue["message"]
                    )
                    if issue["severity"] == "high":
                        exit_code = 1
            
            if table.row_count > 0:
                console.print(table)
            
            if high_severity_count > 0:
                console.print(f"[red]✗ Найдено {high_severity_count} критических проблем[/red]")
                exit_code = 1
            elif len(all_issues) > 0:
                console.print(f"[yellow]⚠ Найдено {len(all_issues)} проблем (все некритические)[/yellow]")
        else:
            console.print("[green]✓ Проблем не найдено[/green]")
    
    # 3. Проверка upstream
    if check_upstream_val:
        console.print(Panel("[bold blue]3. Проверка доступности upstream[/bold blue]", box=box.ROUNDED))
        if tree is None:
            try:
                tree = parse_nginx_config(config_path)
            except FileNotFoundError:
                console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
                sys.exit(1)
            except Exception as e:
                console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
                sys.exit(1)
        
        upstreams = tree.get_upstreams()
        if upstreams:
            upstream_results = check_upstreams(upstreams, timeout=timeout_val, retries=1, mode="tcp", max_workers=max_workers_val)
            
            unhealthy_count = 0
            upstream_servers = []
            
            for name, servers in upstream_results.items():
                for srv in servers:
                    server_info = {
                        "upstream": name,
                        "address": srv["address"],
                        "healthy": srv["healthy"]
                    }
                    upstream_servers.append(server_info)
                    if not srv["healthy"]:
                        unhealthy_count += 1
            
            results["upstream"]["servers"] = upstream_servers
            results["upstream"]["healthy"] = unhealthy_count == 0
            
            if unhealthy_count > 0:
                console.print(f"[red]✗ Найдено {unhealthy_count} недоступных upstream серверов[/red]")
                table = Table(show_header=True, header_style="bold red")
                table.add_column("Upstream")
                table.add_column("Address")
                table.add_column("Status")
                
                for srv in upstream_servers:
                    if not srv["healthy"]:
                        table.add_row(srv["upstream"], srv["address"], "[red]Unhealthy[/red]")
                
                console.print(table)
                exit_code = 1
            else:
                console.print(f"[green]✓ Все upstream серверы доступны ({len(upstream_servers)} серверов)[/green]")
        else:
            console.print("[yellow]⚠ Upstream серверы не найдены[/yellow]")
    
    # 4. Проверка DNS резолвинга
    if check_dns_val:
        console.print(Panel("[bold blue]4. Проверка DNS резолвинга upstream[/bold blue]", box=box.ROUNDED))
        if tree is None:
            try:
                tree = parse_nginx_config(config_path)
            except FileNotFoundError:
                console.print(f"[red]Файл {config_path} не найден. Проверьте путь к конфигу.[/red]")
                sys.exit(1)
            except Exception as e:
                console.print(f"[red]Ошибка при разборе {config_path}: {e}[/red]")
                sys.exit(1)
        
        upstreams = tree.get_upstreams()
        if upstreams:
            dns_results = resolve_upstreams(upstreams, max_workers=max_workers_val, use_cache=True, cache_ttl=cache_ttl)
            
            failed_count = 0
            invalid_count = 0
            dns_servers = []
            
            for name, servers in dns_results.items():
                for srv in servers:
                    server_info = {
                        "upstream": name,
                        "address": srv["address"],
                        "resolved": srv["resolved"],
                        "status": "success"
                    }
                    
                    if not srv["resolved"]:
                        server_info["status"] = "failed"
                        failed_count += 1
                    elif any("invalid resolve" in r for r in srv["resolved"]):
                        server_info["status"] = "invalid"
                        invalid_count += 1
                    
                    dns_servers.append(server_info)
            
            results["dns"]["servers"] = dns_servers
            results["dns"]["resolved"] = (failed_count + invalid_count) == 0
            
            if failed_count > 0 or invalid_count > 0:
                console.print(f"[red]✗ Проблемы с DNS резолвингом: {failed_count} не резолвится, {invalid_count} невалидных[/red]")
                table = Table(show_header=True, header_style="bold red")
                table.add_column("Upstream")
                table.add_column("Address")
                table.add_column("Status")
                
                for srv in dns_servers:
                    if srv["status"] != "success":
                        status_str = "Failed" if srv["status"] == "failed" else "Invalid"
                        resolved_str = ", ".join(srv["resolved"]) if srv["resolved"] else "Failed to resolve"
                        table.add_row(srv["upstream"], srv["address"], f"[red]{status_str}: {resolved_str}[/red]")
                
                console.print(table)
                exit_code = 1
            else:
                console.print(f"[green]✓ Все DNS имена резолвятся корректно ({len(dns_servers)} серверов)[/green]")
        else:
            console.print("[yellow]⚠ Upstream серверы не найдены[/yellow]")
    
    # Экспорт результатов
    if json or yaml:
        from exporter.json_yaml import print_export
        export_data = {
            "timestamp": __import__('datetime').datetime.now().isoformat(),
            "config_path": config_path,
            "results": results,
            "valid": exit_code == 0
        }
        format_type = 'json' if json else 'yaml'
        print_export(export_data, format_type)
        sys.exit(exit_code)
    
    # Итоговый результат
    console.print("")
    if exit_code == 0:
        console.print(Panel("[bold green]✓ Валидация пройдена успешно[/bold green]", box=box.ROUNDED))
    else:
        console.print(Panel("[bold red]✗ Валидация не пройдена. Обнаружены проблемы.[/bold red]", box=box.ROUNDED))
    
    sys.exit(exit_code)


def _check_syntax(config_path: str, nginx_path: str) -> Tuple[bool, List[str]]:
    """
    Проверяет синтаксис конфигурации через nginx -t.
    
    Args:
        config_path: Путь к конфигурационному файлу
        nginx_path: Путь к бинарю nginx
        
    Returns:
        Кортеж (valid, errors) где valid - валидность, errors - список ошибок
    """
    cmd = [nginx_path, "-t", "-c", os.path.abspath(config_path)]
    
    # Если не root, пробуем через sudo
    if hasattr(os, 'geteuid') and os.geteuid() != 0:
        cmd = ["sudo"] + cmd
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=False, timeout=10)
        
        if result.returncode == 0:
            return True, []
        else:
            # Парсим ошибки из stderr
            errors = []
            error_output = result.stderr or result.stdout
            if error_output:
                # Простой парсинг ошибок nginx
                for line in error_output.split('\n'):
                    if 'error' in line.lower() or 'failed' in line.lower():
                        errors.append(line.strip())
            
            return False, errors if errors else [error_output.strip()] if error_output.strip() else ["Неизвестная ошибка синтаксиса"]
    except FileNotFoundError:
        return False, [f"Бинарь nginx не найден: {nginx_path}"]
    except subprocess.TimeoutExpired:
        return False, ["Таймаут при проверке синтаксиса"]
    except Exception as e:
        return False, [f"Ошибка при проверке синтаксиса: {e}"]

