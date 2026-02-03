import sys
from typing import Optional, List, Dict, Any
import typer
from rich.console import Console
from rich.table import Table
import re
import gzip
from datetime import datetime, timedelta
from collections import Counter, defaultdict
from exporter.json_yaml import format_logs_results, print_export
from exporter.csv import export_logs_to_csv
from config.config_loader import get_config

app = typer.Typer(help="Анализ access.log/error.log: топ-статусы, пути, IP, User-Agent, ошибки.")
console = Console()

# Улучшенный regex для парсинга nginx access log (поддерживает response time)
# Формат: IP - - [timestamp] "method path protocol" status size "referer" "user-agent" "response_time"
log_line_re = re.compile(
    r'(?P<ip>\S+) \S+ \S+ \[(?P<time>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) [^\"]+" '
    r'(?P<status>\d{3}) (?P<size>\S+) "(?P<referer>[^"]*)" "(?P<user_agent>[^"]*)"'
    r'(?: "(?P<response_time>[^"]+)")?'
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
    detect_anomalies: bool = typer.Option(False, "--detect-anomalies", help="Обнаруживать аномалии в логах"),
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
        nginx-lens logs /var/log/nginx/access.log --since "2024-01-01" --status 404,500
        nginx-lens logs /var/log/nginx/access.log.gz --detect-anomalies --json
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
    
    # Чтение лога (поддержка gzip)
    try:
        if log_path.endswith('.gz'):
            with gzip.open(log_path, 'rt', encoding='utf-8', errors='ignore') as f:
                lines = list(f)
        else:
            with open(log_path, 'r', encoding='utf-8', errors='ignore') as f:
                lines = list(f)
    except FileNotFoundError:
        console.print(f"[red]Файл {log_path} не найден. Проверьте путь к логу.[/red]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка при чтении {log_path}: {e}[/red]")
        sys.exit(1)
    status_counter = Counter()
    path_counter = Counter()
    ip_counter = Counter()
    user_agent_counter = Counter()
    errors = defaultdict(list)
    response_times = []
    log_entries = []
    
    # Парсинг nginx формата времени: 01/Jan/2024:00:00:00 +0000
    nginx_time_format = "%d/%b/%Y:%H:%M:%S %z"
    
    for line in lines:
        m = log_line_re.search(line)
        if m:
            try:
                # Парсинг времени
                time_str = m.group('time')
                log_time = datetime.strptime(time_str, nginx_time_format)
                
                # Убираем timezone для сравнения (приводим к naive datetime)
                if log_time.tzinfo:
                    log_time = log_time.replace(tzinfo=None)
                
                # Фильтрация по времени
                if since_dt and log_time < since_dt:
                    continue
                if until_dt and log_time > until_dt:
                    continue
                
                ip = m.group('ip')
                path = m.group('path')
                status = m.group('status')
                method = m.group('method')
                user_agent = m.group('user_agent') or ''
                response_time_str = m.group('response_time')
                
                # Фильтрация по статусам
                if status_filter and status not in status_filter:
                    continue
                
                # Сбор данных
                entry = {
                    'time': log_time,
                    'ip': ip,
                    'path': path,
                    'status': status,
                    'method': method,
                    'user_agent': user_agent,
                    'response_time': float(response_time_str) if response_time_str else None
                }
                log_entries.append(entry)
                
                status_counter[status] += 1
                path_counter[path] += 1
                ip_counter[ip] += 1
                
                if user_agent:
                    user_agent_counter[user_agent] += 1
                
                if status.startswith('4') or status.startswith('5'):
                    errors[status].append(path)
                
                if response_time_str:
                    try:
                        response_times.append(float(response_time_str))
                    except ValueError:
                        pass
            except (ValueError, AttributeError) as e:
                # Пропускаем строки с неверным форматом
                continue
    
    # Проверка на пустые результаты
    if not log_entries:
        if json or yaml or csv:
            empty_data = {
                "timestamp": __import__('datetime').datetime.now().isoformat(),
                "summary": {"total_requests": 0},
                "message": "Нет записей, соответствующих фильтрам"
            }
            if csv:
                print("Category,Type,Value,Count\nNo Data,,,,No entries match filters")
            else:
                format_type = 'json' if json else 'yaml'
                print_export(empty_data, format_type)
        else:
            console.print("[yellow]Нет записей, соответствующих указанным фильтрам.[/yellow]")
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
    
    # Показываем статистику по времени ответа
    if response_time_stats:
        table = Table(title="Response Time Statistics", show_header=True, header_style="bold green")
        table.add_column("Metric")
        table.add_column("Value")
        for metric, value in response_time_stats.items():
            if metric != "total_requests_with_time":
                table.add_row(metric.replace("_", " ").title(), f"{value:.3f}s")
            else:
                table.add_row(metric.replace("_", " ").title(), str(int(value)))
        console.print(table)
    
    # Показываем аномалии
    if anomalies:
        table = Table(title="Detected Anomalies", show_header=True, header_style="bold red")
        table.add_column("Type")
        table.add_column("Description")
        table.add_column("Severity")
        for anomaly in anomalies:
            severity_color = {"high": "red", "medium": "orange3", "low": "yellow"}.get(anomaly.get("severity", "low"), "white")
            table.add_row(
                anomaly.get("type", ""),
                anomaly.get("description", ""),
                f"[{severity_color}]{anomaly.get('severity', '')}[/{severity_color}]"
            )
        console.print(table)
    
    # Топ статусов
    table = Table(title="Top HTTP Status Codes", show_header=True, header_style="bold blue")
    table.add_column("Status")
    table.add_column("Count")
    for status, count in status_counter.most_common(top):
        table.add_row(status, str(count))
    console.print(table)
    # Топ путей
    table = Table(title="Top Paths", show_header=True, header_style="bold blue")
    table.add_column("Path")
    table.add_column("Count")
    for path, count in path_counter.most_common(top):
        table.add_row(path, str(count))
    console.print(table)
    # Топ IP
    table = Table(title="Top IPs", show_header=True, header_style="bold blue")
    table.add_column("IP")
    table.add_column("Count")
    for ip, count in ip_counter.most_common(top):
        table.add_row(ip, str(count))
    console.print(table)
    # Топ User-Agent
    if user_agent_counter:
        table = Table(title="Top User-Agents", show_header=True, header_style="bold blue")
        table.add_column("User-Agent")
        table.add_column("Count")
        for ua, count in user_agent_counter.most_common(top):
            table.add_row(ua, str(count))
        console.print(table)
    # Топ 404/500
    for err in ('404', '500'):
        if errors[err]:
            table = Table(title=f"Top {err} Paths", show_header=True, header_style="bold blue")
            table.add_column("Path")
            table.add_column("Count")
            c = Counter(errors[err])
            for path, count in c.most_common(top):
                table.add_row(path, str(count))
            console.print(table) 