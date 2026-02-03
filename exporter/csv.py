"""
Модуль для экспорта результатов в CSV формат.
"""
import csv
import sys
from typing import List, Dict, Any
from io import StringIO


def export_logs_to_csv(
    status_counter,
    path_counter,
    ip_counter,
    user_agent_counter,
    errors: Dict[str, List[str]],
    response_times: List[Dict[str, Any]] = None,
    anomalies: List[Dict[str, Any]] = None
) -> str:
    """
    Экспортирует результаты анализа логов в CSV формат.
    
    Args:
        status_counter: Счетчик статусов
        path_counter: Счетчик путей
        ip_counter: Счетчик IP
        user_agent_counter: Счетчик User-Agent
        errors: Словарь ошибок по статусам
        response_times: Список данных о времени ответа
        anomalies: Список аномалий
        
    Returns:
        CSV строка
    """
    output = StringIO()
    writer = csv.writer(output)
    
    # Топ статусов
    writer.writerow(["Category", "Type", "Value", "Count"])
    writer.writerow(["Status Codes", "", "", ""])
    for status, count in status_counter.most_common():
        writer.writerow(["", "Status", status, count])
    
    writer.writerow([])
    writer.writerow(["Paths", "", "", ""])
    for path, count in path_counter.most_common():
        writer.writerow(["", "Path", path, count])
    
    writer.writerow([])
    writer.writerow(["IPs", "", "", ""])
    for ip, count in ip_counter.most_common():
        writer.writerow(["", "IP", ip, count])
    
    if user_agent_counter:
        writer.writerow([])
        writer.writerow(["User-Agents", "", "", ""])
        for ua, count in user_agent_counter.most_common():
            writer.writerow(["", "User-Agent", ua, count])
    
    # Ошибки
    if errors:
        writer.writerow([])
        writer.writerow(["Errors", "", "", ""])
        for status, paths in errors.items():
            writer.writerow(["", f"Error {status}", f"{len(paths)} occurrences", ""])
    
    # Response times
    if response_times:
        writer.writerow([])
        writer.writerow(["Response Times", "", "", ""])
        writer.writerow(["", "Metric", "Value", ""])
        for metric, value in response_times.items():
            writer.writerow(["", metric, str(value), ""])
    
    # Аномалии
    if anomalies:
        writer.writerow([])
        writer.writerow(["Anomalies", "Type", "Description", "Severity"])
        for anomaly in anomalies:
            writer.writerow([
                "",
                anomaly.get("type", ""),
                anomaly.get("description", ""),
                anomaly.get("severity", "")
            ])
    
    return output.getvalue()

