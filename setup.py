from setuptools import setup, find_packages

setup(
    name="nginx-lens",
    version="0.3.2",
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
    ],
    entry_points={
        "console_scripts": [
            "nginx-lens=commands.cli:app",
        ],
    },
    python_requires=">=3.8",
) 