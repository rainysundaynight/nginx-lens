"""
Команда для генерации скриптов автодополнения.
"""
import sys
import typer
from rich.console import Console
from rich.panel import Panel
from rich import box

app = typer.Typer()
console = Console()


def _generate_completion_script(shell: str) -> str:
    """
    Генерирует базовый completion скрипт для указанного shell.
    
    Args:
        shell: Тип shell (bash, zsh, fish, powershell)
        
    Returns:
        Строка с completion скриптом
    """
    commands = [
        "health", "analyze", "tree", "diff", "route", "include",
        "graph", "logs", "syntax", "resolve", "validate", "metrics"
    ]
    
    if shell == "bash":
        script = f"""# Bash completion for nginx-lens
_nginx_lens_completion() {{
    local cur="${{COMP_WORDS[COMP_CWORD]}}"
    local commands="{' '.join(commands)}"
    
    if [[ ${{COMP_CWORD}} -eq 1 ]]; then
        COMPREPLY=($(compgen -W "$commands" -- "$cur"))
    else
        COMPREPLY=($(compgen -f -- "$cur"))
    fi
}}

complete -F _nginx_lens_completion nginx-lens
"""
    elif shell == "zsh":
        script = f"""# Zsh completion for nginx-lens
#compdef nginx-lens

_nginx_lens() {{
    local -a commands
    commands=(
        {' '.join(f'"{cmd}":{cmd}' for cmd in commands)}
    )
    
    _describe 'command' commands
}}

_nginx_lens "$@"
"""
    elif shell == "fish":
        script = f"""# Fish completion for nginx-lens
complete -c nginx-lens -f -a "{' '.join(commands)}"
"""
    elif shell == "powershell":
        script = f"""# PowerShell completion for nginx-lens
Register-ArgumentCompleter -Native -CommandName nginx-lens -ScriptBlock {{
    param($commandName, $wordToComplete, $cursorPosition)
    
    $commands = @({', '.join(f'"{cmd}"' for cmd in commands)})
    
    $commands | Where-Object {{ $_ -like "$wordToComplete*" }} | ForEach-Object {{
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }}
}}
"""
    else:
        script = f"# Completion script for {shell}\n# Use: typer commands.cli app {shell}\n"
    
    return script


@app.command()
def install(
    shell: str = typer.Argument(..., help="Тип shell: bash, zsh, fish, powershell"),
    output: str = typer.Option(None, "--output", "-o", help="Путь для сохранения скрипта (по умолчанию выводится в stdout)"),
):
    """
    Генерирует скрипт автодополнения для указанного shell.
    
    После генерации нужно добавить его в конфигурацию shell:
    
    Bash:
        nginx-lens completion install bash >> ~/.bashrc
    
    Zsh:
        nginx-lens completion install zsh >> ~/.zshrc
    
    Fish:
        nginx-lens completion install fish > ~/.config/fish/completions/nginx-lens.fish
    
    PowerShell:
        nginx-lens completion install powershell > nginx-lens-completion.ps1
        # Затем выполните: . .\nginx-lens-completion.ps1
    """
    import subprocess
    import sys
    
    # Пробуем использовать typer CLI для генерации completion
    completion_script = None
    try:
        result = subprocess.run(
            [sys.executable, "-m", "typer", "commands.cli", "app", shell],
            capture_output=True,
            text=True,
            check=False,
            timeout=10
        )
        
        if result.returncode == 0 and result.stdout and len(result.stdout.strip()) > 0:
            completion_script = result.stdout
    except (subprocess.TimeoutExpired, FileNotFoundError, Exception):
        pass
    
    # Если typer CLI не сработал, используем базовый скрипт
    if not completion_script:
        completion_script = _generate_completion_script(shell)
    
    if shell not in ["bash", "zsh", "fish", "powershell"]:
        console.print(f"[red]Неподдерживаемый shell: {shell}[/red]")
        console.print("[yellow]Поддерживаемые shell: bash, zsh, fish, powershell[/yellow]")
        raise typer.Exit(1)
    
    if output:
        try:
            with open(output, 'w') as f:
                f.write(completion_script)
            console.print(f"[green]✓ Скрипт автодополнения сохранен в {output}[/green]")
            console.print(f"[yellow]Не забудьте добавить его в конфигурацию {shell}[/yellow]")
        except Exception as e:
            console.print(f"[red]Ошибка при сохранении файла: {e}[/red]")
            raise typer.Exit(1)
        else:
            typer.echo(completion_script)
            # Выводим инструкцию в stderr, чтобы не мешать скрипту
            console.print("\n[yellow]Добавьте этот скрипт в конфигурацию вашего shell[/yellow]", file=sys.stderr)


@app.command()
def show_instructions():
    """
    Показывает инструкции по установке автодополнения для всех поддерживаемых shell.
    """
    console.print(Panel("[bold blue]Инструкции по установке автодополнения[/bold blue]", box=box.ROUNDED))
    
    console.print("\n[bold]Bash:[/bold]")
    console.print("  nginx-lens completion install bash >> ~/.bashrc")
    console.print("  source ~/.bashrc")
    
    console.print("\n[bold]Zsh:[/bold]")
    console.print("  nginx-lens completion install zsh >> ~/.zshrc")
    console.print("  source ~/.zshrc")
    
    console.print("\n[bold]Fish:[/bold]")
    console.print("  mkdir -p ~/.config/fish/completions")
    console.print("  nginx-lens completion install fish > ~/.config/fish/completions/nginx-lens.fish")
    
    console.print("\n[bold]PowerShell:[/bold]")
    console.print("  nginx-lens completion install powershell > nginx-lens-completion.ps1")
    console.print("  . .\\nginx-lens-completion.ps1")
    console.print("  # Или добавьте в профиль PowerShell: $PROFILE")


if __name__ == "__main__":
    app()

