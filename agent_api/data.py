from __future__ import annotations

from typing import Any, Dict, Optional

from analyzer.conflicts import (
    find_listen_servername_conflicts,
    find_location_conflicts,
)
from analyzer.dead_locations import find_dead_locations
from analyzer.duplicates import find_duplicate_directives
from analyzer.empty_blocks import find_empty_blocks
from analyzer.rewrite import find_rewrite_issues
from analyzer.unused import find_unused_variables
from analyzer.warnings import find_warnings
from commands.analyze import ISSUE_META
from config.config_loader import get_config
from exporter.json_yaml import format_analyze_results
from parser.nginx_parser import parse_nginx_config
from upstream_checker.checker import check_upstreams


def _resolve_config_path(config_path: Optional[str]) -> str:
    if config_path:
        return config_path

    cfg = get_config()
    p = cfg.get_nginx_config_path()
    if not p:
        raise ValueError(
            "nginx config path is not provided (config_path) and not set in config file"
        )
    return p


def collect_snapshot(config_path: Optional[str] = None) -> Dict[str, Any]:
    """
    Collects a JSON-serializable snapshot of nginx config analysis.

    This intentionally reuses existing CLI analyzers to keep results consistent
    between CLI and the agent API.
    """
    cfg = get_config()
    defaults = cfg.get_defaults()
    cache_config = cfg.get_cache_config()
    dynamic_upstream_config = cfg.get_dynamic_upstream_config()

    path = _resolve_config_path(config_path)

    tree = parse_nginx_config(path)
    tree.set_dynamic_upstream_config(
        dynamic_upstream_config.get("enabled", False),
        dynamic_upstream_config.get("api_url"),
        dynamic_upstream_config.get("timeout", 2.0),
    )

    # Analyze (same building blocks as `nginx-lens analyze`)
    conflicts = find_location_conflicts(tree.directives)
    dups = find_duplicate_directives(tree.directives)
    empties = find_empty_blocks(tree.directives)
    warnings = find_warnings(tree.directives)
    unused_vars = find_unused_variables(tree.directives)
    listen_conflicts = find_listen_servername_conflicts(tree.directives)
    rewrite_issues = find_rewrite_issues(tree.directives)
    dead_locations = find_dead_locations(tree.directives)

    analyze_export = format_analyze_results(
        conflicts=conflicts,
        dups=dups,
        empties=empties,
        warnings=warnings,
        unused_vars=unused_vars,
        listen_conflicts=listen_conflicts,
        rewrite_issues=rewrite_issues,
        dead_locations=dead_locations,
        issue_meta=ISSUE_META,
    )

    # Upstreams
    upstreams = tree.get_upstreams()
    timeout = float(defaults.get("timeout", 2.0))
    retries = int(defaults.get("retries", 1))
    mode = str(defaults.get("mode", "tcp"))
    max_workers = int(defaults.get("max_workers", 10))

    # Respect cache config (same behavior as CLI: it toggles global DNS cache),
    # but for the agent snapshot we only include the config in response.
    cache_info = {
        "enabled": bool(cache_config.get("enabled", True)),
        "ttl": cache_config.get("ttl", defaults.get("dns_cache_ttl", 300)),
    }

    health = check_upstreams(
        upstreams=upstreams,
        timeout=timeout,
        retries=retries,
        mode=mode,
        max_workers=max_workers,
        progress_manager=None,
    )

    return {
        "config_path": path,
        "cache": cache_info,
        "dynamic_upstream": dynamic_upstream_config,
        "analyze": analyze_export,
        "upstreams": upstreams,
        "upstreams_health": health,
    }

