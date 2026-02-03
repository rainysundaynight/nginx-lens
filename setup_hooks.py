"""
Хуки для автоматической установки автодополнения при установке пакета.
"""
import os
import sys
from pathlib import Path
from typing import Optional

try:
    import yaml
except ImportError:
    yaml = None


def detect_shell() -> Optional[str]:
    """
    Определяет текущий shell пользователя.
    
    Returns:
        Имя shell (bash, zsh, fish, powershell) или None
    """
    shell = os.environ.get('SHELL', '')
    
    if 'bash' in shell.lower():
        return 'bash'
    elif 'zsh' in shell.lower():
        return 'zsh'
    elif 'fish' in shell.lower():
        return 'fish'
    elif 'powershell' in shell.lower() or 'pwsh' in shell.lower():
        return 'powershell'
    
    # Дополнительная проверка через ps
    try:
        import subprocess
        result = subprocess.run(['ps', '-p', str(os.getppid()), '-o', 'comm='], 
                              capture_output=True, text=True, timeout=2)
        if result.returncode == 0:
            shell_name = result.stdout.strip()
            if 'bash' in shell_name:
                return 'bash'
            elif 'zsh' in shell_name:
                return 'zsh'
            elif 'fish' in shell_name:
                return 'fish'
    except Exception:
        pass
    
    return None


def get_completion_script(shell: str) -> str:
    """
    Генерирует completion скрипт для указанного shell.
    
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
        return ""
    
    return script


def install_completion(shell: str, dry_run: bool = False) -> bool:
    """
    Устанавливает completion скрипт для указанного shell.
    
    Args:
        shell: Тип shell (bash, zsh, fish, powershell)
        dry_run: Если True, только проверяет возможность установки
        
    Returns:
        True если установка успешна, False иначе
    """
    script = get_completion_script(shell)
    if not script:
        return False
    
    home = Path.home()
    
    try:
        if shell == "bash":
            # Устанавливаем в ~/.bashrc
            bashrc = home / ".bashrc"
            if dry_run:
                return bashrc.exists() or (home / ".bash_profile").exists()
            
            # Проверяем, не установлено ли уже
            if bashrc.exists():
                with open(bashrc, 'r') as f:
                    if 'nginx-lens' in f.read():
                        return True  # Уже установлено
            
            # Добавляем в .bashrc
            with open(bashrc, 'a') as f:
                f.write(f"\n# nginx-lens completion\n{script}\n")
            return True
            
        elif shell == "zsh":
            # Устанавливаем в ~/.zshrc
            zshrc = home / ".zshrc"
            if dry_run:
                return zshrc.exists()
            
            # Проверяем, не установлено ли уже
            if zshrc.exists():
                with open(zshrc, 'r') as f:
                    if 'nginx-lens' in f.read():
                        return True  # Уже установлено
            
            # Добавляем в .zshrc
            with open(zshrc, 'a') as f:
                f.write(f"\n# nginx-lens completion\n{script}\n")
            return True
            
        elif shell == "fish":
            # Устанавливаем в ~/.config/fish/completions/nginx-lens.fish
            completions_dir = home / ".config" / "fish" / "completions"
            completion_file = completions_dir / "nginx-lens.fish"
            
            if dry_run:
                return completions_dir.exists() or completions_dir.parent.exists()
            
            # Создаем директорию если нужно
            completions_dir.mkdir(parents=True, exist_ok=True)
            
            # Записываем completion файл
            with open(completion_file, 'w') as f:
                f.write(script)
            return True
            
        elif shell == "powershell":
            # Устанавливаем в профиль PowerShell
            try:
                import subprocess
                result = subprocess.run(
                    ['powershell', '-Command', '$PROFILE'],
                    capture_output=True,
                    text=True,
                    timeout=5
                )
                if result.returncode == 0:
                    profile_path = Path(result.stdout.strip())
                    if dry_run:
                        return profile_path.parent.exists()
                    
                    # Создаем директорию профиля если нужно
                    profile_path.parent.mkdir(parents=True, exist_ok=True)
                    
                    # Проверяем, не установлено ли уже
                    if profile_path.exists():
                        with open(profile_path, 'r') as f:
                            if 'nginx-lens' in f.read():
                                return True  # Уже установлено
                    
                    # Добавляем в профиль
                    with open(profile_path, 'a') as f:
                        f.write(f"\n# nginx-lens completion\n{script}\n")
                    return True
            except Exception:
                pass
            
            return False
    except Exception:
        return False
    
    return False


def create_default_config():
    """
    Создает директорию /opt/nginx-lens и дефолтный конфиг.
    
    Returns:
        True если успешно, False иначе
    """
    if yaml is None:
        return False
    
    opt_dir = Path("/opt/nginx-lens")
    config_file = opt_dir / "config.yaml"
    
    # Дефолтный конфиг
    default_config = {
        "defaults": {
            "timeout": 2.0,
            "retries": 1,
            "mode": "tcp",
            "max_workers": 10,
            "dns_cache_ttl": 300,
            "top": 10,
        },
        "output": {
            "colors": True,
            "format": "table",
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
    
    try:
        # Создаем директорию если нужно
        if not opt_dir.exists():
            opt_dir.mkdir(parents=True, mode=0o755, exist_ok=True)
        
        # Проверяем, существует ли уже конфиг
        if config_file.exists():
            # Конфиг уже существует, не перезаписываем
            return True
        
        # Создаем дефолтный конфиг
        with open(config_file, 'w') as f:
            yaml.dump(default_config, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
        
        # Устанавливаем права доступа
        config_file.chmod(0o644)
        
        return True
    except PermissionError:
        # Нет прав для создания в /opt, это нормально
        return False
    except Exception as e:
        # Другая ошибка
        return False


def post_install():
    """
    Выполняется после установки пакета для автоматической установки completion и создания конфига.
    """
    # Создаем дефолтный конфиг в /opt/nginx-lens
    try:
        if create_default_config():
            print("✓ Конфигурационный файл создан в /opt/nginx-lens/config.yaml")
        else:
            print("⚠ Не удалось создать конфиг в /opt/nginx-lens (требуются права root)")
            print("  Конфиг можно создать вручную: nginx-lens completion show-instructions")
    except Exception as e:
        print(f"⚠ Ошибка при создании конфига: {e}")
    
    # Определяем shell
    shell = detect_shell()
    
    if not shell:
        # Не можем определить shell, пропускаем установку
        return
    
    # Пробуем установить completion
    try:
        if install_completion(shell, dry_run=False):
            print(f"✓ Автодополнение для {shell} установлено автоматически")
            if shell in ['bash', 'zsh']:
                print(f"  Выполните 'source ~/.{shell}rc' для применения изменений")
        else:
            print(f"⚠ Не удалось автоматически установить автодополнение для {shell}")
            print(f"  Выполните вручную: nginx-lens completion install {shell}")
    except Exception as e:
        print(f"⚠ Ошибка при установке автодополнения: {e}")
        print(f"  Выполните вручную: nginx-lens completion install {shell}")

