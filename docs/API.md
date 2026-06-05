# API

JSON-схемы для CI и HTTP API agent/hub. CLI читает настройки из [config.yaml](CONFIGURATION.md).

## CLI: Analyze

```json
{
  "issues": [
    {
      "type": "ssl_missing",
      "description": "on",
      "solution": "Укажите SSL-сертификат/ключ.",
      "severity": "high",
      "file": "/etc/nginx/conf.d/ssl.conf",
      "line": 12,
      "fix_hint": "ssl_certificate /path/cert.pem;\nssl_certificate_key /path/key.pem;"
    }
  ],
  "summary": { "high": 1, "medium": 0, "low": 0 }
}
```

Включить: `output.format: json` → `nginx-lens analyze`.

## CLI: Health

```json
{
  "upstreams": {
    "backend": [{ "address": "127.0.0.1:8080", "healthy": true }]
  }
}
```

## CLI: Resolve

```json
{
  "upstreams": {
    "backend": [{ "address": "example.com:80", "resolved": ["1.2.3.4:80"] }]
  }
}
```

## CLI: Explain

```json
{
  "url": "http://example.com/api/v1",
  "server": { "block": "server", "file": "/etc/nginx/nginx.conf" },
  "location": { "block": "location", "arg": "/api" },
  "proxy_pass": "http://api_backend",
  "upstream": "api_backend",
  "trace": [
    { "step": "server", "detail": "server_name=example.com", "matched": true },
    { "step": "location", "detail": "выбран =/api", "matched": true }
  ]
}
```

## CLI: Blast-radius

```json
{
  "upstream": "api_backend",
  "healthy": false,
  "impact": [
    {
      "upstream_name": "api_backend",
      "server_name": "example.com",
      "location": "/api",
      "from_directive": "proxy_pass",
      "config_file": "/etc/nginx/conf.d/api.conf"
    }
  ]
}
```

## CLI: Certs

```json
[
  {
    "type": "cert_expiring",
    "severity": "medium",
    "cert_path": "/etc/ssl/cert.pem",
    "days_left": 14,
    "message": "истекает через 14 дней"
  }
]
```

Типы: `cert_expired`, `cert_expiring`, `cert_self_signed`, `cert_hostname_mismatch`, `cert_chain_incomplete`, `cert_not_found`, `cert_invalid_pem`.

## CLI: Score

```json
{
  "total": 78.5,
  "categories": [
    { "name": "security", "score": 85, "weight": 0.3, "issues": 2 }
  ],
  "top_actions": ["ssl_protocols_weak: Отключите устаревшие TLS."]
}
```

## CLI: Validate

Объект с ключами `syntax`, `analysis`, `upstream`, `dns`, `certs`, `policy`, `fail_on`. Exit code 1 — см. [CONFIGURATION.md](CONFIGURATION.md#validate).

## Agent HTTP

| Endpoint | Auth | Ответ |
|----------|------|-------|
| `GET /healthz` | нет | `{"ok": true}` |
| `GET /version` | нет | `{"version": "2.3.0"}` |
| `GET /metrics` | нет | Prometheus text |
| `GET /snapshot` | token | Snapshot JSON |

### Snapshot (фрагмент)

```json
{
  "config_path": "/etc/nginx/nginx.conf",
  "analyze": { "issues": [], "summary": {} },
  "upstreams": { "api_backend": ["10.0.0.1:8080"] },
  "upstreams_health": {
    "api_backend": [{ "address": "10.0.0.1:8080", "healthy": false }]
  },
  "score": { "total": 82, "categories": [], "top_actions": [] },
  "policy_issues": [],
  "cert_issues": [],
  "error_stats": {
    "total_errors": 847,
    "upstream_errors": [{ "upstream": "http://api_backend", "count": 847 }]
  },
  "meta": {
    "hostname": "nginx-host-1",
    "timestamp": "2026-06-05T12:00:00Z",
    "agent_version": "2.3.0"
  }
}
```

`error_stats` заполняется при `logs.error_path` на агенте.

Auth: `X-Nginx-Lens-Token: <web.agent.token>`.

## Hub HTTP

| Endpoint | Auth | Описание |
|----------|------|----------|
| `GET /` | нет | Dashboard |
| `GET /api/v1/status` | hub token | Статус агентов |
| `GET /api/v1/snapshots` | hub token | Snapshots |

### Snapshots

```json
{
  "results": [
    {
      "agent": "http://10.0.0.5:8088",
      "online": true,
      "snapshot": {}
    }
  ]
}
```

Auth: `X-Nginx-Lens-Token: <web.hub.token>`.

Деплой — [WEB_DEPLOYMENT.md](WEB_DEPLOYMENT.md).
