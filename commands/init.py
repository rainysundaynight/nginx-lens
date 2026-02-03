"""
Команда для инициализации nginx-lens (создание конфига и установка completion).
"""
import sys
import typer
from rich.console import Console
from rich.panel import Panel
from rich import box

app = typer.Typer(help="Инициализация nginx-lens: создание конфига и установка автодополнения.")
console = Console()


@app.command()
def init(
    force: bool = typer.Option(False, "--force", "-f", help="Перезаписать существующий конфиг")
):
    """
    Инициализирует nginx-lens: создает конфиг и устанавливает автодополнение.
    
    Эта команда выполняет те же действия, что должны выполняться автоматически
    при установке пакета через pip. Если автоматическая установка не сработала,
    выполните эту команду вручную.
    
    Примеры:
        nginx-lens init
        nginx-lens init --force
    """
    try:
        from utils.init_helpers import post_install, create_default_config, detect_shell, install_completion
        
        # Создаем конфиг
        console.print("[cyan]Создание конфигурационного файла...[/cyan]")
        success, error_msg = create_default_config()
        if success:
            console.print("[green]✓ Конфигурационный файл создан в /opt/nginx-lens/config.yaml[/green]")
        else:
            if "Permission denied" in error_msg or "Нет прав" in error_msg:
                console.print(f"[yellow]⚠ Не удалось создать конфиг в /opt/nginx-lens: {error_msg}[/yellow]")
                console.print("[yellow]  Попробуйте выполнить с правами root:[/yellow]")
                console.print("[yellow]    sudo nginx-lens init[/yellow]")
                console.print("[yellow]  Или создайте конфиг вручную в ~/.nginx-lens/config.yaml[/yellow]")
            else:
                console.print(f"[red]✗ Ошибка при создании конфига: {error_msg}[/red]")
        
        # Устанавливаем completion
        console.print("\n[cyan]Установка автодополнения...[/cyan]")
        shell = detect_shell()
        if not shell:
            console.print("[yellow]⚠ Не удалось определить shell. Пропускаем установку автодополнения.[/yellow]")
            console.print("[yellow]  Выполните вручную: nginx-lens completion install <shell>[/yellow]")
        else:
            try:
                if install_completion(shell, dry_run=False):
                    console.print(f"[green]✓ Автодополнение для {shell} установлено[/green]")
                    if shell in ['bash', 'zsh']:
                        console.print(f"[yellow]  Выполните 'source ~/.{shell}rc' для применения изменений[/yellow]")
                else:
                    console.print(f"[yellow]⚠ Не удалось установить автодополнение для {shell}[/yellow]")
                    console.print(f"[yellow]  Выполните вручную: nginx-lens completion install {shell}[/yellow]")
            except Exception as e:
                console.print(f"[red]✗ Ошибка при установке автодополнения: {e}[/red]")
                console.print(f"[yellow]  Выполните вручную: nginx-lens completion install {shell}[/yellow]")
        
        console.print("\n[green]✓ Инициализация завершена![/green]")
        
    except ImportError as e:
        console.print(f"[red]✗ Ошибка импорта: {e}[/red]")
        console.print("[yellow]  Убедитесь, что nginx-lens установлен корректно.[/yellow]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]✗ Неожиданная ошибка: {e}[/red]")
        sys.exit(1)

