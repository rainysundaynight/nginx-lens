"""Версия дистрибутива. После `pip install` берётся из metadata; в dev-дереве — FALLBACK."""


FALLBACK_VERSION = "0.9.2"


def get_version() -> str:
    from importlib.metadata import version, PackageNotFoundError

    try:
        return version("nginx-lens")
    except PackageNotFoundError:
        return FALLBACK_VERSION
