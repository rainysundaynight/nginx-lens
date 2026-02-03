"""
Модуль для загрузки и управления конфигурационным файлом nginx-lens.
"""
import os
import yaml
from pathlib import Path
from typing import Optional, Dict, Any


def _find_default_nginx_config() -> Optional[str]:
    """
    Ищет nginx.conf в стандартных местах.
    
    Returns:
        Путь к nginx.conf или None
    """
    candidates = [
        "/etc/nginx/nginx.conf",
        "/usr/local/etc/nginx/nginx.conf",
        "./nginx.conf",
    ]
    for path in candidates:
        if os.path.exists(path) and os.path.isfile(path):
            return path
    return None


class ConfigLoader:
    """
    Загрузчик конфигурационного файла для nginx-lens.
    """
    
    def __init__(self):
        """Инициализация загрузчика конфигурации."""
        self.config: Dict[str, Any] = {}
        self.config_path: Optional[Path] = None
        self._load_config()
    
    def _find_config_file(self) -> Optional[Path]:
        """
        Ищет конфигурационный файл в стандартных местах.
        
        Returns:
            Path к конфигурационному файлу или None
        """
        # Список возможных путей к конфигу (в порядке приоритета)
        possible_paths = [
            Path.cwd() / ".nginx-lens.yaml",  # Текущая директория
            Path.cwd() / ".nginx-lens.yml",
            Path("/opt/nginx-lens/config.yaml"),  # Системный конфиг
            Path("/opt/nginx-lens/config.yml"),
            Path.home() / ".nginx-lens" / "config.yaml",  # Домашняя директория
            Path.home() / ".nginx-lens" / "config.yml",
        ]
        
        for path in possible_paths:
            if path.exists() and path.is_file():
                return path
        
        return None
    
    def _load_config(self):
        """Загружает конфигурацию из файла."""
        config_file = self._find_config_file()
        
        if not config_file:
            # Используем значения по умолчанию
            self.config = self._get_default_config()
            return
        
        try:
            with open(config_file, 'r') as f:
                self.config = yaml.safe_load(f) or {}
            
            # Объединяем с дефолтными значениями
            default_config = self._get_default_config()
            self.config = self._merge_config(default_config, self.config)
            self.config_path = config_file
        except (yaml.YAMLError, IOError) as e:
            # В случае ошибки используем дефолтные значения
            self.config = self._get_default_config()
    
    def _get_default_config(self) -> Dict[str, Any]:
        """
        Возвращает конфигурацию по умолчанию.
        
        Returns:
            Словарь с дефолтными настройками
        """
        return {
            "defaults": {
                "timeout": 2.0,
                "retries": 1,
                "mode": "tcp",
                "max_workers": 10,
                "dns_cache_ttl": 300,
                "top": 10,
                "nginx_config_path": None,  # Путь к nginx.conf (если None - используется автопоиск)
            },
            "output": {
                "colors": True,
                "format": "table",  # table, json, yaml
            },
            "cache": {
                "enabled": True,
                "ttl": 300,
            },
            "validate": {
                "check_syntax": True,
                "check_analysis": True,
                "check_upstream": True,
                "check_dns": False,
                "nginx_path": "nginx",
            }
        }
    
    def _merge_config(self, default: Dict[str, Any], user: Dict[str, Any]) -> Dict[str, Any]:
        """
        Рекурсивно объединяет конфигурации.
        
        Args:
            default: Конфигурация по умолчанию
            user: Пользовательская конфигурация
            
        Returns:
            Объединенная конфигурация
        """
        result = default.copy()
        
        for key, value in user.items():
            if key in result and isinstance(result[key], dict) and isinstance(value, dict):
                result[key] = self._merge_config(result[key], value)
            else:
                result[key] = value
        
        return result
    
    def get(self, section: str, key: str, default: Any = None) -> Any:
        """
        Получает значение из конфигурации.
        
        Args:
            section: Секция конфига (например, "defaults", "output")
            key: Ключ в секции
            default: Значение по умолчанию, если не найдено
            
        Returns:
            Значение из конфига или default
        """
        return self.config.get(section, {}).get(key, default)
    
    def get_defaults(self) -> Dict[str, Any]:
        """
        Получает все значения по умолчанию.
        
        Returns:
            Словарь с настройками по умолчанию
        """
        return self.config.get("defaults", {})
    
    def get_output_config(self) -> Dict[str, Any]:
        """
        Получает настройки вывода.
        
        Returns:
            Словарь с настройками вывода
        """
        return self.config.get("output", {})
    
    def get_cache_config(self) -> Dict[str, Any]:
        """
        Получает настройки кэша.
        
        Returns:
            Словарь с настройками кэша
        """
        return self.config.get("cache", {})
    
    def get_validate_config(self) -> Dict[str, Any]:
        """
        Получает настройки команды validate.
        
        Returns:
            Словарь с настройками validate
        """
        return self.config.get("validate", {})
    
    def get_nginx_config_path(self) -> Optional[str]:
        """
        Получает путь к nginx.conf из конфигурации или автопоиск.
        
        Returns:
            Путь к nginx.conf или None
        """
        path = self.config.get("defaults", {}).get("nginx_config_path")
        if path:
            # Проверяем существование файла
            if os.path.exists(path) and os.path.isfile(path):
                return path
        # Если не указан в конфиге, пробуем автопоиск
        return _find_default_nginx_config()
    
    def get_config_path(self) -> Optional[str]:
        """
        Возвращает путь к загруженному конфигурационному файлу.
        
        Returns:
            Путь к конфигу или None
        """
        return str(self.config_path) if self.config_path else None


# Глобальный экземпляр загрузчика конфигурации
_config_loader: Optional[ConfigLoader] = None


def get_config() -> ConfigLoader:
    """
    Получает глобальный экземпляр загрузчика конфигурации.
    
    Returns:
        Экземпляр ConfigLoader
    """
    global _config_loader
    if _config_loader is None:
        _config_loader = ConfigLoader()
    return _config_loader


def reload_config():
    """Перезагружает конфигурацию."""
    global _config_loader
    _config_loader = ConfigLoader()

