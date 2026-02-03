from setuptools import setup, find_packages
from setuptools.command.install import install
import sys


class PostInstallCommand(install):
    """
    Кастомная команда установки для автоматической установки автодополнения.
    """
    def run(self):
        # Выполняем стандартную установку
        install.run(self)
        
        # Устанавливаем автодополнение
        try:
            from setup_hooks import post_install
            post_install()
        except Exception as e:
            # Не критично, если не удалось установить completion
            print(f"⚠ Не удалось установить автодополнение автоматически: {e}")
            print("  Выполните вручную: nginx-lens completion show-instructions")


setup(
    name="nginx-lens",
    version="0.5.2",
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
    },
) 