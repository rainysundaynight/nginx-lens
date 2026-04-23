"""
Сбор ссылок на upstream (proxy_pass, fastcgi_pass, …) и разбор server-строк.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any, Dict, Iterable, List, Optional, Set

__all__ = [
    "UpstreamRef",
    "iter_all_directives",
    "iter_upstream_blocks",
    "collect_known_upstream_names",
    "find_upstream_references",
    "parse_server_options",
    "host_from_http_style_url",
    "host_from_grpc_url",
]


@dataclass
class UpstreamRef:
    """Ссылка на named upstream из server/location/stream."""

    upstream_name: str
    from_directive: str
    value: str
    server_name: str
    listen: str
    location: str
    config_file: str
    is_stream: bool = False

    def location_display(self) -> str:
        return self.location if self.location and self.location != "—" else "—"


def iter_all_directives(directives: List[Any]) -> Iterable[Dict[str, Any]]:
    for d in directives:
        yield d
        subs = d.get("directives")
        if subs:
            yield from iter_all_directives(subs)


def iter_upstream_blocks(directives: List[Any]) -> Iterable[Dict[str, Any]]:
    for d in iter_all_directives(directives):
        if d.get("upstream"):
            yield d


def collect_known_upstream_names(directives: List[Any]) -> Set[str]:
    return {d["upstream"] for d in iter_upstream_blocks(directives) if d.get("upstream")}


def _parse_host_port(token: str) -> str:
    if not token or token.startswith("$"):
        return token
    if token.startswith("["):
        m = re.match(r"^(\[[^\]]+\])(?::(\d+))?$", token)
        return m.group(1) if m else token
    if re.match(r"^[\d.]+$", token) or re.match(r"^[\d.]+:\d+$", token):
        if token.count(":") == 1:
            a, b = token.rsplit(":", 1)
            if b.isdigit():
                return a
    if token.count(":") == 1:
        a, b = token.rsplit(":", 1)
        if b.isdigit():
            return a
    return token


def host_from_http_style_url(s: str) -> Optional[str]:
    s = s.strip()
    if not s.startswith("http://") and not s.startswith("https://"):
        return None
    rest = s.split("://", 1)[1]
    slash = rest.find("/")
    if slash != -1:
        rest = rest[:slash]
    if not rest or rest.startswith("$"):
        return None
    return _parse_host_port(rest)


def host_from_grpc_url(s: str) -> Optional[str]:
    s = s.strip()
    for pfx in ("grpc://", "grpcs://"):
        if s.startswith(pfx):
            rest = s[len(pfx) :]
            rest = rest.split("/", 1)[0]
            if not rest or rest.startswith("$"):
                return None
            return _parse_host_port(rest)
    return None


def _resolve_ref(directive: str, args: str, known: Set[str]) -> Optional[str]:
    d = (directive or "").lower()
    a = (args or "").strip()
    if d == "proxy_pass":
        if a.startswith("http://") or a.startswith("https://"):
            h = host_from_http_style_url(a)
            return h if h and h in known else None
        if a in known:
            return a
    if d in ("fastcgi_pass", "uwsgi_pass", "scgi_pass"):
        if a in known:
            return a
    if d == "memcached_pass" and a in known:
        return a
    if d == "grpc_pass":
        h = host_from_grpc_url(a)
        if h and h in known:
            return h
    return None


def _read_server_name_listen(directives: List[Dict]) -> (str, str):
    names: List[str] = []
    listens: List[str] = []
    for s in directives:
        if s.get("directive") == "server_name":
            names += (s.get("args") or "").split()
        if s.get("directive") == "listen":
            listens.append((s.get("args") or "").strip())
    sn = " ".join(names) if names else "—"
    ls = ", ".join(listens) if listens else "—"
    return sn, ls


def find_upstream_references(
    top_directives: List[Dict], known: Set[str]
) -> List[UpstreamRef]:
    out: List[UpstreamRef] = []

    def go(
        items: List[Dict],
        in_stream: bool,
        server_name: str,
        listen: str,
        loc: str,
        cfile: str,
    ) -> None:
        for d in items:
            f = d.get("__file__") or cfile or "—"
            b = d.get("block")
            if b == "http":
                go(d.get("directives", []), False, "—", "—", "—", f)
            elif b == "stream":
                go(d.get("directives", []), True, "—", "—", "—", f)
            elif b == "server":
                subs = d.get("directives", [])
                sn, ls = _read_server_name_listen(subs)
                go(subs, in_stream, sn, ls, "—", f)
            elif b == "location":
                nloc = d.get("arg") or ""
                go(d.get("directives", []), in_stream, server_name, listen, nloc, f)
            elif b == "upstream":
                continue
            elif b:
                go(
                    d.get("directives", []), in_stream, server_name, listen, loc, f
                )
            if d.get("directive"):
                name = d.get("directive", "")
                val = d.get("args", "") or ""
                u = _resolve_ref(name, val, known)
                if u:
                    out.append(
                        UpstreamRef(
                            upstream_name=u,
                            from_directive=name,
                            value=val,
                            server_name=server_name,
                            listen=listen,
                            location=loc or "—",
                            config_file=f,
                            is_stream=in_stream,
                        )
                    )

    go(top_directives, False, "—", "—", "—", "")
    return out


# --- server-строка: адрес + флаги (weight, backup, ...) ---

_SRV_TOK = re.compile(r"(\w+)=(\S+)")


def parse_server_options(server_line: str) -> Dict[str, str]:
    s = (server_line or "").strip()
    if not s:
        return {
            "address": "—",
            "raw": "—",
            "weight": "—",
            "max_fails": "—",
            "fail_timeout": "—",
            "backup": "—",
            "down": "—",
            "other": "—",
        }
    parts = s.split()
    address = parts[0]
    rest = parts[1:]
    out = {
        "address": address,
        "raw": s,
        "weight": "—",
        "max_fails": "—",
        "fail_timeout": "—",
        "backup": "—",
        "down": "—",
        "other": "—",
    }
    flags = []
    for p in rest:
        pl = p.lower()
        if pl == "backup":
            out["backup"] = "да"
        elif pl == "down":
            out["down"] = "да"
        else:
            m = _SRV_TOK.match(p)
            if m:
                k, v = m.group(1).lower(), m.group(2)
                if k == "weight":
                    out["weight"] = v
                elif k == "max_fails":
                    out["max_fails"] = v
                elif k == "fail_timeout":
                    out["fail_timeout"] = v
                else:
                    flags.append(f"{k}={v}")
            else:
                flags.append(p)
    if flags:
        out["other"] = " ".join(flags) if len(flags) < 4 else " ".join(flags[:3]) + "…"
    return out
