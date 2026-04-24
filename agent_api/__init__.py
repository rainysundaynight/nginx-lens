"""nginx-lens agent API package."""

__all__ = ["__version__"]

try:
    from utils.version import get_version as _get_version

    __version__ = _get_version()
except Exception:
    __version__ = "0.0.0"

