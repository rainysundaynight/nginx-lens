"""
Утилиты для отображения прогресса операций.
"""
from typing import Optional, Callable, Any
from rich.progress import Progress, SpinnerColumn, BarColumn, TextColumn, TimeElapsedColumn, TimeRemainingColumn, TaskID
from rich.console import Console
import signal
import sys

console = Console()


class ProgressManager:
    """
    Менеджер для отображения прогресса операций с поддержкой отмены.
    """
    
    def __init__(self, description: str = "Обработка", show_progress: bool = True):
        """
        Инициализация менеджера прогресса.
        
        Args:
            description: Описание операции
            show_progress: Показывать ли прогресс-бар
        """
        self.description = description
        self.show_progress = show_progress
        self.progress: Optional[Progress] = None
        self.task_id: Optional[TaskID] = None
        self.cancelled = False
        
        # Обработка сигнала прерывания
        self._original_sigint = signal.signal(signal.SIGINT, self._handle_interrupt)
    
    def _handle_interrupt(self, signum, frame):
        """Обработка прерывания (Ctrl+C)."""
        self.cancelled = True
        if self.progress:
            self.progress.stop()
        console.print("\n[yellow]Операция отменена пользователем[/yellow]")
        sys.exit(130)  # Стандартный exit code для SIGINT
    
    def __enter__(self):
        """Вход в контекстный менеджер."""
        if self.show_progress:
            self.progress = Progress(
                SpinnerColumn(),
                TextColumn("[progress.description]{task.description}"),
                BarColumn(),
                TextColumn("[progress.percentage]{task.percentage:>3.0f}%"),
                TimeElapsedColumn(),
                TimeRemainingColumn(),
                console=console,
                transient=False,
            )
            self.progress.start()
            self.task_id = self.progress.add_task(self.description, total=None)
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Выход из контекстного менеджера."""
        if self.progress:
            self.progress.stop()
        # Восстанавливаем оригинальный обработчик сигнала
        signal.signal(signal.SIGINT, self._original_sigint)
        return False
    
    def update(self, completed: int, total: Optional[int] = None, description: Optional[str] = None):
        """
        Обновляет прогресс.
        
        Args:
            completed: Количество завершенных задач
            total: Общее количество задач (если None, используется спиннер)
            description: Описание текущей операции
        """
        if self.progress and self.task_id is not None:
            if total is not None:
                self.progress.update(self.task_id, total=total, completed=completed)
            if description:
                self.progress.update(self.task_id, description=description)
    
    def advance(self, advance: int = 1):
        """
        Увеличивает прогресс на указанное значение.
        
        Args:
            advance: На сколько увеличить прогресс
        """
        if self.progress and self.task_id is not None:
            self.progress.advance(self.task_id, advance)


def with_progress(
    description: str,
    total: Optional[int] = None,
    show_progress: bool = True
) -> Callable:
    """
    Декоратор для добавления прогресс-бара к функции.
    
    Args:
        description: Описание операции
        total: Общее количество итераций (если известно)
        show_progress: Показывать ли прогресс-бар
        
    Returns:
        Декорированная функция
    """
    def decorator(func: Callable) -> Callable:
        def wrapper(*args, **kwargs):
            with ProgressManager(description=description, show_progress=show_progress) as pm:
                # Передаем менеджер прогресса в функцию
                kwargs['progress_manager'] = pm
                if total is not None:
                    pm.update(0, total=total)
                return func(*args, **kwargs)
        return wrapper
    return decorator

