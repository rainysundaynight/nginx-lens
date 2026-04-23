"""
Парсинг access-логов: nginx combined, JSONL, LTSV, эвристика, пользовательский regex.
"""
from __future__ import annotations

import json
import re
from dataclasses import dataclass
from datetime import datetime
from typing import Any, Dict, Optional, Pattern, Tuple
from urllib.parse import urlparse

# --- nginx combined ---
_NGX_COMBINED = re.compile(
    r'(?P<ip>\S+)\s+\S+\s+\S+\s+\[(?P<time>[^\]]+)\]\s+'
    r'"(?P<method>\S+)\s+(?P<path>\S+)\s+[^"]+"\s+'
    r'(?P<status>\d{3})\s+(?P<size>\S+)\s+'
    r'"(?P<referer>[^"]*)"\s+"(?P<user_agent>[^"]*)"'
    r'(?:\s+"(?P<response_time>[^"]*)")?'
)
_NGX_MINIMAL = re.compile(
    r'(?P<ip>\S+)\s+\S+\s+\S+\s+\[(?P<time>[^\]]+)\]\s+'
    r'"(?P<method>\S+)\s+(?P<path>\S+)\s+[^"]+"\s+'
    r'(?P<status>\d{3})\s+(?P<size>\S+)'
)

_QREQ = re.compile(
    r'"(?P<method>GET|POST|HEAD|PUT|DELETE|PATCH|OPTIONS|CONNECT|TRACE|PROPPATCH|PROPFIND|MKCOL|COPY|MOVE)\s+'
    r'(?P<path>[^\s?"]+)(?:\?[^"]*?)?\s+HTTP/[\d.]+?"',
    re.IGNORECASE,
)
_UREQ = re.compile(
    r'(?<![A-Za-z])(?P<method>GET|POST|PUT|DELETE|HEAD|PATCH|OPTIONS|TRACE|CONNECT)\s+'
    r'(?P<path>/[^\s]+|https?://\S+)',
    re.IGNORECASE,
)
_BRACKET_TIME = re.compile(
    r"\[(\d{1,2}/[A-Za-z]{3}/\d{4}:\d{2}:\d{2}:\d{2} [^]]+)\]"
)
_NGX_TIME = "%d/%b/%Y:%H:%M:%S %z"
_ISO_STRPTIME: Tuple[str, ...] = (
    "%Y-%m-%dT%H:%M:%S.%f%z",
    "%Y-%m-%dT%H:%M:%S%z",
    "%Y-%m-%dT%H:%M:%S.%fZ",
    "%Y-%m-%dT%H:%M:%SZ",
    "%Y-%m-%d %H:%M:%S%z",
    "%Y-%m-%d %H:%M:%S",
)


@dataclass
class ParsedLogLine:
    time: Optional[datetime]  # naive
    method: str
    path: str
    status: str
    ip: str
    user_agent: str
    response_time: Optional[float]


def _to_naive(dt: Optional[datetime]) -> Optional[datetime]:
    if dt is None:
        return None
    if dt.tzinfo:
        return dt.replace(tzinfo=None)
    return dt


def _parse_nginx_time(s: str) -> Optional[datetime]:
    s = s.strip()
    if not s:
        return None
    try:
        return datetime.strptime(s, _NGX_TIME)
    except ValueError:
        pass
    for fmt in _ISO_STRPTIME:
        if fmt.endswith("Z") and not s.endswith("Z"):
            continue
        try:
            return datetime.strptime(s, fmt)
        except ValueError:
            pass
    if "T" in s:
        try:
            s2 = s.replace("Z", "+00:00")
            d = datetime.fromisoformat(s2)
            return d
        except ValueError:
            pass
    try:
        t = float(s)
        if 1e9 < t < 1e11:
            return datetime.utcfromtimestamp(t)
    except (ValueError, OSError, OverflowError):
        pass
    return None


def _parse_time_any(raw: Any) -> Optional[datetime]:
    if raw is None:
        return None
    if isinstance(raw, (int, float)):
        t = float(raw)
        if 1e9 < t < 1e14:
            if t > 1e12:
                t /= 1000.0
            return datetime.utcfromtimestamp(t)
    if not isinstance(raw, str):
        raw = str(raw)
    if not raw or raw in ("0", "null", "None"):
        return None
    if raw.isdigit() and 1e9 < int(raw) < 1e15:
        t = int(raw)
        if t > 1e12:
            t //= 1000
        return datetime.utcfromtimestamp(t)
    return _parse_nginx_time(raw)


def _parse_request_str(req: str) -> Tuple[Optional[str], str]:
    req = (req or "").strip()
    if not req:
        return None, "/"
    parts = req.split()
    if len(parts) >= 2 and not parts[0].startswith("/") and (
        parts[1].startswith("/") or parts[1].lower().startswith("http")
    ):
        return parts[0].upper(), parts[1]
    return None, parts[0] if parts else "/"


def _get_ci(d: Dict[str, Any], *candidates: str) -> Any:
    """Первый непустой ключ (регистронезависимый)."""
    if not d:
        return None
    lower = {k.lower(): k for k in d if isinstance(k, str)}
    for c in candidates:
        if c in d:
            v = d[c]
        elif c.lower() in lower:
            v = d[lower[c.lower()]]
        else:
            continue
        if v is not None and v != "":
            return v
    return None


def _int_status(v: Any) -> Optional[str]:
    if v is None:
        return None
    if isinstance(v, int):
        s = v
    else:
        try:
            s = int(str(v).split()[0])
        except (ValueError, TypeError, IndexError):
            return None
    if 100 <= s <= 599:
        return str(s)
    return None


def _from_flat_dict(d: dict) -> Optional[ParsedLogLine]:
    st = _int_status(_get_ci(d, "status", "http_status", "status_code", "response_status", "responseStatus", "sc"))

    # GCP httpRequest
    hreq = d.get("httpRequest")
    if isinstance(hreq, dict):
        p = hreq.get("requestUrl") or hreq.get("url")
        if p and st is None:
            st = _int_status(hreq.get("status"))
        m = hreq.get("requestMethod")
        if p:
            u = urlparse(p)
            path = u.path or "/"
        else:
            path = None
        method = (m or "GET")
        t = _parse_time_any(
            d.get("timestamp") or d.get("@timestamp") or hreq.get("requestTime")
        )
        ua = str(hreq.get("userAgent") or "")
        ip = str(
            d.get("remote_ip")
            or hreq.get("remoteIp")
            or _get_ci(d, "ip", "remote_addr", "client_ip")
            or "unknown"
        )
        if path and st:
            return ParsedLogLine(
                time=_to_naive(t), method=method, path=path, status=str(st), ip=ip, user_agent=ua, response_time=None
            )
    path: Optional[str] = _get_ci(d, "path", "request_uri", "uri", "requestPath", "url")
    if isinstance(path, str) and (path.isdigit() and len(path) == 3):
        path = None
    if isinstance(path, str) and (path.startswith("http://") or path.startswith("https://")):
        path = urlparse(path).path or "/"

    method: Optional[str] = _get_ci(d, "method", "http_method", "verb", "requestMethod")
    req = d.get("request")
    if isinstance(req, str) and (path is None or (method is None and " " in req)):
        m, p2 = _parse_request_str(req)
        if p2 and (p2.startswith("/") or p2.lower().startswith("http")):
            if path is None:
                path = urlparse(p2).path if p2.lower().startswith("http") else p2.split("?")[0]
            if m and method is None:
                method = m
        sm = re.search(r"HTTP/[\d.]+\s*(\d{3})", req)
        if st is None and sm:
            st = sm.group(1)
    if isinstance(req, dict):
        p = req.get("path")
        rurl = req.get("requestUrl")
        if p is None and isinstance(rurl, str):
            p = urlparse(rurl).path or "/"
        if p and not path:
            path = p
        if st is None:
            st = _int_status(req.get("status"))
        if not method and req.get("requestMethod"):
            method = req["requestMethod"]
    if path is not None and not isinstance(path, str):
        path = str(path)
    if path and " " in path and path.split() and not path.lstrip().startswith("{"):
        m, p2 = _parse_request_str(path)
        if p2 and (p2.startswith("/") or p2.lower().startswith("http")):
            path = p2.split("?")[0] if p2.startswith("/") else (urlparse(p2).path or "/")
        if m and not method:
            method = m

    if (path is None or st is None) and isinstance(d.get("message"), str):
        from_msg = _parse_loose(d["message"])
        if from_msg is not None:
            return from_msg
    st = _int_status(st) or None
    if not path or not st:
        return None
    t = _parse_time_any(
        _get_ci(d, "time", "timestamp", "@timestamp", "date", "time_local", "log_time", "start_time", "t")
    )
    ua = str(_get_ci(d, "user_agent", "userAgent", "http_user_agent", "User-Agent", "ua", "useragent") or "")
    ip = str(_get_ci(d, "ip", "remote_addr", "client_ip", "X-Forwarded-For", "x_forwarded_for", "source_ip") or "unknown")
    if "," in ip:
        ip = ip.split(",")[0].strip()
    rt: Optional[float] = None
    for k in d:
        if k in ("request_time", "response_time", "duration", "latency", "took", "rt") or k.lower() in {
            "request_time", "response_time", "duration", "latency"
        }:
            try:
                rt = float(d[k])  # type: ignore[index]
            except (TypeError, ValueError, KeyError):
                pass
            if rt is not None:
                break
    if rt is not None and rt > 1e6:
        rt /= 1e6
    mth = (method or "—").split()[0].upper() if method else "—"
    return ParsedLogLine(
        time=_to_naive(t),
        method=mth,
        path=path,
        status=str(st),
        ip=ip,
        user_agent=ua,
        response_time=rt,
    )


def _parse_nginx_combined_line(line: str) -> Optional[ParsedLogLine]:
    m = _NGX_COMBINED.search(line) or _NGX_MINIMAL.search(line)
    if not m:
        return None
    g = m.groupdict()
    ts = g.get("time")
    if not ts:
        return None
    log_time = _parse_nginx_time(ts) or _parse_time_any(ts)
    if log_time is None:
        return None
    log_time = _to_naive(log_time)
    rt: Optional[float] = None
    raw = g.get("response_time")
    if raw:
        try:
            rt = float(str(raw).replace("ms", "").strip())
        except ValueError:
            pass
    return ParsedLogLine(
        time=log_time,
        method=(g.get("method") or "—").split()[0].upper(),
        path=g.get("path") or "/",
        status=str(g.get("status")),
        ip=g.get("ip") or "unknown",
        user_agent=(g.get("user_agent") or "") or "",
        response_time=rt,
    )


def _parse_json_line(line: str) -> Optional[ParsedLogLine]:
    s = line.strip()
    if not s or s[0] != "{":
        return None
    try:
        obj: Any = json.loads(s)
    except json.JSONDecodeError:
        return None
    if not isinstance(obj, dict):
        return None
    return _from_flat_dict(obj)


def _parse_ltsv_line(line: str) -> Optional[ParsedLogLine]:
    if "\t" not in line:
        return None
    d: Dict[str, str] = {}
    for p in line.rstrip().split("\t"):
        if not p or ":" not in p:
            continue
        k, v = p.split(":", 1)
        d[k.strip()] = v
    if not d:
        return None
    return _from_flat_dict(d)


def _first_ip_ish(line: str) -> str:
    m = re.match(r"^(\S+)", line)
    if not m:
        return "unknown"
    t = m.group(1)
    if re.match(r"^(\d{1,3}\.){3}\d{1,3}$", t):
        return t
    if t.startswith("["):
        return t[1:].rstrip("]") or "unknown"
    return t if t.replace(":", "").replace("::", "x").isalnum() or ":" in t else "unknown"


def _parse_loose(line: str) -> Optional[ParsedLogLine]:
    m = _QREQ.search(line)
    method = path = None
    rest = line
    if m:
        method, path = m.group("method"), m.group("path")
        rest = line[m.end() :]
    if not m:
        u = _UREQ.search(line)
        if u:
            method, path = u.group("method"), u.group("path")
            if path.startswith("http"):
                path = urlparse(path).path or "/"
            else:
                path = path.split("?")[0]
            rest = line[u.end() :]
    if not method or not path:
        return None
    mst = re.search(r"(^|[^0-9])([1-5][0-9][0-9])([^0-9]|$)", rest)
    if not mst or not (100 <= int(mst.group(2)) < 600):
        return None
    status = mst.group(2)
    t_m = _BRACKET_TIME.search(line)
    var_time: Optional[datetime] = None
    if t_m:
        var_time = _to_naive(_parse_nginx_time(t_m.group(1)) or _parse_time_any(t_m.group(1)))
    return ParsedLogLine(
        time=var_time,
        method=method.upper(),
        path=path,
        status=status,
        ip=_first_ip_ish(line),
        user_agent="",
        response_time=None,
    )


def _parse_custom(line: str, custom_re: Pattern[str]) -> Optional[ParsedLogLine]:
    cm = custom_re.search(line)
    if not cm:
        return None
    gd = cm.groupdict()
    p = (gd.get("path") or "").strip() if isinstance(gd.get("path"), (str, type(None))) else str(gd.get("path", ""))
    st = (gd.get("status") or "").strip() if isinstance(gd.get("status"), (str, type(None))) else str(gd.get("status", ""))
    p = p or ""
    st = st or ""
    if not p or not re.match(r"^\d{3}$", st) or not (100 <= int(st) < 600):
        return None
    th = (gd.get("time") or "").strip() or None
    t: Optional[datetime] = None
    if th:
        t = _to_naive(_parse_nginx_time(th) or _parse_time_any(th))
    method = gd.get("method") or "—"
    if isinstance(method, str) and method:
        method = method.split()[0].upper()
    else:
        method = "—"
    ip = (gd.get("ip") or "unknown")
    if isinstance(ip, str):
        ip = ip.split(",")[0].strip() or "unknown"
    else:
        ip = "unknown"
    ua = (gd.get("user_agent") or gd.get("ua") or "") or ""
    rtf: Optional[float] = None
    rtr = gd.get("response_time")
    if rtr not in (None, ""):
        try:
            rtf = float(str(rtr).replace("ms", ""))
        except ValueError:
            pass
    return ParsedLogLine(
        time=t, method=method, path=p, status=st, ip=ip, user_agent=ua, response_time=rtf
    )


def parse_log_line(
    line: str,
    mode: str,
    custom_re: Optional[Pattern[str]] = None,
) -> Optional[ParsedLogLine]:
    line = line.rstrip("\n\r")
    if not line.strip():
        return None
    m = (mode or "auto").lower()
    if custom_re is not None:
        return _parse_custom(line, custom_re)

    if m in ("auto", "nginx", "combined", "nginx-combined"):
        r = _parse_nginx_combined_line(line)
        if r is not None:
            return r
        if m in ("nginx", "combined", "nginx-combined"):
            return None
    if m in ("auto", "json", "jsonl"):
        r = _parse_json_line(line)
        if r is not None:
            return r
        if m in ("json", "jsonl"):
            return None
    if m in ("auto", "ltsv"):
        r = _parse_ltsv_line(line)
        if r is not None:
            return r
        if m == "ltsv":
            return None
    if m in ("auto", "loose", "heuristic"):
        r = _parse_loose(line)
        if r is not None:
            return r
        if m in ("loose", "heuristic"):
            return None
    if m in ("any", "flex", "universal", "all", "try-all"):
        for fn in (
            _parse_nginx_combined_line,
            _parse_json_line,
            _parse_ltsv_line,
            _parse_loose,
        ):
            r = fn(line)
            if r is not None:
                return r
        return None
    return None


def parse_universal_log_line(line: str) -> Optional[ParsedLogLine]:
    """Точка входа для CLI: комбинированный разбор без настроек пользователя."""
    return parse_log_line(line, "auto", None)
