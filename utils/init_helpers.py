"""
Вспомогательные функции для инициализации nginx-lens.
Используются как при установке пакета, так и при выполнении команды init.
"""
import os
import sys
from pathlib import Path
from typing import Optional, Tuple

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
        "graph", "logs", "syntax", "resolve", "validate", "metrics", "init"
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


def create_default_config() -> Tuple[bool, Optional[str]]:
    """
    Создает директорию /opt/nginx-lens и дефолтный конфиг.
    
    Returns:
        Tuple[bool, Optional[str]]: (успешно ли, сообщение об ошибке или None)
    """
    if yaml is None:
        return False, "PyYAML не установлен"
    
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
            "nginx_config_path": None,  # Путь к nginx.conf (если None - используется автопоиск)
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
            try:
                opt_dir.mkdir(parents=True, mode=0o755, exist_ok=True)
            except PermissionError as e:
                return False, f"Нет прав для создания директории /opt/nginx-lens (требуются права root): {e}"
            except OSError as e:
                return False, f"Ошибка при создании директории /opt/nginx-lens: {e}"
        
        # Проверяем, существует ли уже конфиг
        if config_file.exists():
            # Конфиг уже существует, не перезаписываем
            return True, None
        
        # Создаем дефолтный конфиг
        try:
            with open(config_file, 'w') as f:
                yaml.dump(default_config, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
        except PermissionError as e:
            return False, f"Нет прав для создания файла {config_file} (требуются права root): {e}"
        except IOError as e:
            return False, f"Ошибка записи в файл {config_file}: {e}"
        
        # Устанавливаем права доступа
        try:
            config_file.chmod(0o644)
        except Exception:
            # Не критично, если не удалось установить права
            pass
        
        return True, None
    except Exception as e:
        return False, f"Неожиданная ошибка: {e}"


def post_install():
    """
    Выполняется после установки пакета для автоматической установки completion и создания конфига.
    """
    # Создаем дефолтный конфиг в /opt/nginx-lens
    try:
        success, error_msg = create_default_config()
        if success:
            print("✓ Конфигурационный файл создан в /opt/nginx-lens/config.yaml", file=sys.stderr)
        else:
            print(f"⚠ Не удалось создать конфиг в /opt/nginx-lens: {error_msg}", file=sys.stderr)
            print("  Для создания конфига выполните:", file=sys.stderr)
            print("    sudo mkdir -p /opt/nginx-lens", file=sys.stderr)
            print("    sudo nginx-lens init", file=sys.stderr)
            print("  Или создайте конфиг вручную в ~/.nginx-lens/config.yaml", file=sys.stderr)
    except Exception as e:
        print(f"⚠ Ошибка при создании конфига: {e}", file=sys.stderr)
    
    # Определяем shell
    shell = detect_shell()
    
    if not shell:
        # Не можем определить shell, пропускаем установку
        return
    
    # Пробуем установить completion
    try:
        if install_completion(shell, dry_run=False):
            print(f"✓ Автодополнение для {shell} установлено автоматически", file=sys.stderr)
            if shell in ['bash', 'zsh']:
                print(f"  Выполните 'source ~/.{shell}rc' для применения изменений", file=sys.stderr)
        else:
            print(f"⚠ Не удалось автоматически установить автодополнение для {shell}", file=sys.stderr)
            print(f"  Выполните вручную: nginx-lens completion install {shell}", file=sys.stderr)
    except Exception as e:
        print(f"⚠ Ошибка при установке автодополнения: {e}", file=sys.stderr)
        print(f"  Выполните вручную: nginx-lens completion install {shell}", file=sys.stderr)

