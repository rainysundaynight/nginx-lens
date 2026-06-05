# Web-стек: Agent + Hub

Распределённый мониторинг: агенты на nginx-хостах отдают snapshot, Hub агрегирует данные, **встроенный dashboard** доступен на `GET /`.

## Компоненты

| Бинарник | Назначение | Порт |
|----------|------------|------|
| `nginx-lens-agent` | Snapshot конфига + health на каждом хосте | 8088 |
| `nginx-lens-hub` | Агрегация агентов + UI | 8089 |

Настройки — секция `web:` в [config.yaml](CONFIGURATION.md). Запуск без CLI-флагов:

```bash
nginx-lens-agent
nginx-lens-hub
```

## Agent

### config.yaml

```yaml
defaults:
  nginx_config_path: /etc/nginx/nginx.conf
logs:
  error_path: /var/log/nginx/error.log
web:
  agent:
    host: 127.0.0.1
    port: 8088
    token: "change-me"
```

### systemd

```ini
[Unit]
Description=nginx-lens agent
After=network.target nginx.service

[Service]
Environment=NGINX_LENS_CONFIG=/opt/nginx-lens/config.yaml
ExecStart=/usr/local/bin/nginx-lens-agent
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Endpoints

| Path | Auth | Описание |
|------|------|----------|
| `GET /healthz` | нет | Liveness |
| `GET /version` | нет | Версия |
| `GET /metrics` | нет | Prometheus metrics |
| `GET /snapshot` | token | Полный snapshot |

Заголовок: `X-Nginx-Lens-Token: <web.agent.token>` или `Authorization: Bearer …`.

## Hub

### config.yaml

```yaml
web:
  hub:
    host: 0.0.0.0
    port: 8089
    token: "hub-secret"
    agent_token: "agent-secret"
    agents:
      - http://10.0.0.5:8088
      - http://10.0.0.6:8088
    cors_origins:
      - "https://dashboard.example.com"
    refresh_interval: 30
```

- `agent_token` — токен Hub → Agent (если пуст, используется `web.agent.token`)
- `agents` — список URL агентов

### Endpoints

| Path | Auth | Описание |
|------|------|----------|
| `GET /` | нет | Dashboard UI |
| `GET /healthz` | нет | Liveness |
| `GET /api/v1/status` | hub token | Online/offline агентов |
| `GET /api/v1/snapshots` | hub token | Агрегированные snapshot |
| `GET /api/v1/explain` | hub token | Explain route (query `url`) |

Legacy-маршруты API: `/api/agents`, `/api/status`, `/api/snapshots`.

JSON-схемы — [API.md](API.md).

## Безопасность

1. Не публикуйте agent/hub в интернет без TLS и auth.
2. Snapshot содержит структуру конфигурации — ограничьте доступ token'ом.
3. Agent bind на `127.0.0.1` + reverse proxy с TLS — рекомендуемый паттерн.
4. Hub CORS: задайте конкретные origin вместо `*` в production.

```nginx
server {
    listen 443 ssl;
    server_name lens.example.com;
    location / {
        proxy_pass http://127.0.0.1:8089;
        proxy_set_header Host $host;
    }
}
```

## Troubleshooting

| Проблема | Решение |
|----------|---------|
| Agent offline в Hub | Firewall, URL в `web.hub.agents`, token |
| 401 Unauthorized | `X-Nginx-Lens-Token` или Bearer token |
| CORS в custom UI | `web.hub.cors_origins` |
| Пустой error_stats | Задайте `logs.error_path` на агенте |

## Установка бинарников

```bash
make build
sudo install bin/nginx-lens-agent bin/nginx-lens-hub /usr/local/bin/
```

Или через [GitHub Releases](https://github.com/rainysundaynight/nginx-lens/releases).
