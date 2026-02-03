"""
Модуль для кэширования результатов DNS резолвинга.
"""
import os
import json
import time
import hashlib
from typing import Optional, List, Dict
from pathlib import Path


class DNSCache:
    """
    Кэш для результатов DNS резолвинга.
    """
    
    def __init__(self, cache_dir: Optional[str] = None, ttl: int = 300):
        """
        Инициализация кэша.
        
        Args:
            cache_dir: Директория для хранения кэша (по умолчанию ~/.cache/nginx-lens/)
            ttl: Время жизни кэша в секундах (по умолчанию 5 минут)
        """
        self.ttl = ttl
        
        if cache_dir:
            self.cache_dir = Path(cache_dir)
        else:
            # Используем ~/.cache/nginx-lens/ или /tmp/nginx-lens-cache/
            home_cache = Path.home() / ".cache" / "nginx-lens"
            tmp_cache = Path("/tmp") / "nginx-lens-cache"
            
            # Пробуем использовать домашнюю директорию, если доступна
            try:
                home_cache.mkdir(parents=True, exist_ok=True)
                self.cache_dir = home_cache
            except (OSError, PermissionError):
                # Fallback на /tmp
                try:
                    tmp_cache.mkdir(parents=True, exist_ok=True)
                    self.cache_dir = tmp_cache
                except (OSError, PermissionError):
                    # Последний fallback - текущая директория
                    self.cache_dir = Path.cwd() / ".nginx-lens-cache"
                    self.cache_dir.mkdir(parents=True, exist_ok=True)
        
        self.cache_file = self.cache_dir / "dns_cache.json"
        self._cache: Dict[str, Dict] = {}
        self._load_cache()
    
    def _get_cache_key(self, host: str, port: str) -> str:
        """
        Генерирует ключ кэша для host:port.
        
        Args:
            host: Имя хоста
            port: Порт
            
        Returns:
            Хеш ключ для кэша
        """
        key = f"{host}:{port}"
        return hashlib.md5(key.encode()).hexdigest()
    
    def _load_cache(self):
        """Загружает кэш из файла."""
        if self.cache_file.exists():
            try:
                with open(self.cache_file, 'r') as f:
                    self._cache = json.load(f)
            except (json.JSONDecodeError, IOError):
                self._cache = {}
    
    def _save_cache(self):
        """Сохраняет кэш в файл."""
        try:
            with open(self.cache_file, 'w') as f:
                json.dump(self._cache, f)
        except IOError:
            # Игнорируем ошибки записи
            pass
    
    def get(self, host: str, port: str) -> Optional[List[str]]:
        """
        Получает результат из кэша.
        
        Args:
            host: Имя хоста
            port: Порт
            
        Returns:
            Список резолвленных IP-адресов или None, если нет в кэше или истек TTL
        """
        key = self._get_cache_key(host, port)
        
        if key not in self._cache:
            return None
        
        cached_data = self._cache[key]
        cached_time = cached_data.get('timestamp', 0)
        current_time = time.time()
        
        # Проверяем TTL
        if current_time - cached_time > self.ttl:
            # Удаляем устаревшую запись
            del self._cache[key]
            self._save_cache()
            return None
        
        return cached_data.get('result')
    
    def set(self, host: str, port: str, result: List[str]):
        """
        Сохраняет результат в кэш.
        
        Args:
            host: Имя хоста
            port: Порт
            result: Список резолвленных IP-адресов
        """
        key = self._get_cache_key(host, port)
        
        self._cache[key] = {
            'timestamp': time.time(),
            'result': result,
            'host': host,
            'port': port
        }
        
        self._save_cache()
    
    def clear(self):
        """Очищает весь кэш."""
        self._cache = {}
        if self.cache_file.exists():
            try:
                self.cache_file.unlink()
            except IOError:
                pass
    
    def get_cache_info(self) -> Dict[str, any]:
        """
        Возвращает информацию о кэше.
        
        Returns:
            Словарь с информацией о кэше
        """
        current_time = time.time()
        valid_entries = 0
        expired_entries = 0
        
        for key, data in self._cache.items():
            cached_time = data.get('timestamp', 0)
            if current_time - cached_time <= self.ttl:
                valid_entries += 1
            else:
                expired_entries += 1
        
        return {
            'total_entries': len(self._cache),
            'valid_entries': valid_entries,
            'expired_entries': expired_entries,
            'cache_dir': str(self.cache_dir),
            'ttl': self.ttl
        }


# Глобальный экземпляр кэша (будет инициализирован при первом использовании)
_cache_instance: Optional[DNSCache] = None
_cache_enabled = True


def get_cache(ttl: int = 300, cache_dir: Optional[str] = None) -> DNSCache:
    """
    Получает глобальный экземпляр кэша.
    
    Args:
        ttl: Время жизни кэша в секундах
        cache_dir: Директория для кэша
        
    Returns:
        Экземпляр DNSCache
    """
    global _cache_instance
    if _cache_instance is None:
        _cache_instance = DNSCache(cache_dir=cache_dir, ttl=ttl)
    elif ttl != _cache_instance.ttl:
        # Обновляем TTL если изменился
        _cache_instance.ttl = ttl
    return _cache_instance


def clear_cache():
    """Очищает глобальный кэш."""
    global _cache_instance
    if _cache_instance:
        _cache_instance.clear()


def disable_cache():
    """Отключает кэширование."""
    global _cache_enabled
    _cache_enabled = False


def enable_cache():
    """Включает кэширование."""
    global _cache_enabled
    _cache_enabled = True


def is_cache_enabled() -> bool:
    """Проверяет, включено ли кэширование."""
    return _cache_enabled

