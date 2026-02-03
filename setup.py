from setuptools import setup, find_packages
from setuptools.command.install import install
from setuptools.command.install_lib import install_lib
import sys
import os


class PostInstallCommand(install):
    """
    Кастомная команда установки для автоматической установки автодополнения.
    """
    def run(self):
        # Выполняем стандартную установку
        install.run(self)
        
        # Устанавливаем автодополнение и создаем конфиг
        self._run_post_install()
    
    def _run_post_install(self):
        """Выполняет пост-установочные действия."""
        try:
            from setup_hooks import post_install
            post_install()
        except Exception as e:
            # Не критично, если не удалось установить completion
            print(f"⚠ Не удалось установить автодополнение автоматически: {e}", file=sys.stderr)
            print("  Выполните вручную: nginx-lens init", file=sys.stderr)


class PostInstallLibCommand(install_lib):
    """
    Альтернативный способ выполнения пост-установочных действий через install_lib.
    Это вызывается даже при wheel установке.
    """
    def run(self):
        # Выполняем стандартную установку библиотек
        install_lib.run(self)
        
        # Выполняем пост-установочные действия
        try:
            # Пробуем импортировать из установленного пакета
            try:
                from utils.init_helpers import post_install
            except ImportError:
                # Fallback на локальный модуль (при первой установке)
                from setup_hooks import post_install
            post_install()
        except Exception as e:
            # Не критично, если не удалось установить completion
            print(f"⚠ Не удалось установить автодополнение автоматически: {e}", file=sys.stderr)
            print("  Выполните вручную: nginx-lens init", file=sys.stderr)


setup(
    name="nginx-lens",
    version="0.5.6",
    description="CLI-инструмент для анализа, визуализации и диагностики конфигураций Nginx",
    author="Daniil Astrouski",
    author_email="shelovesuastra@gmail.com",
    packages=find_packages(),
    include_package_data=True,
    install_requires=[
        "typer[all]>=0.9.0",
        "rich>=13.0.0",
        "requests>=2.25.0",
        "dnspython>=2.0.0",
        "pyyaml>=5.4",
    ],
    extras_require={
        "dev": [
            "pytest>=7.0.0",
            "pytest-cov>=4.0.0",
        ],
    },
    entry_points={
        "console_scripts": [
            "nginx-lens=commands.cli:app",
        ],
    },
    python_requires=">=3.8",
    cmdclass={
        'install': PostInstallCommand,
        'install_lib': PostInstallLibCommand,
    },
) 