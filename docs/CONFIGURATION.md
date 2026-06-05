# Конфигурация

Все команды читают **config.yaml**. Проверка схемы:

```bash
nginx-lens config validate
```

Образец: [example-config.yaml](../example-config.yaml).

## Расположение файла

| Приоритет | Путь |
|-----------|------|
| 1 | `NGINX_LENS_CONFIG` |
| 2 | `/opt/nginx-lens/config.yaml` |
| 3 | `.nginx-lens.yaml` / `.nginx-lens.yml` (cwd) |
| 4 | `~/.nginx-lens/config.yaml` |

## Переменные окружения

Префикс `NGINX_LENS_`, вложенность через `_`:

| Env | Config |
|-----|--------|
| `NGINX_LENS_CONFIG` | путь к config.yaml |
| `NGINX_LENS_DEFAULTS_NGINX_CONFIG_PATH` | `defaults.nginx_config_path` |
| `NGINX_LENS_VALIDATE_FAIL_ON` | `validate.fail_on` |
| `NGINX_LENS_OUTPUT_FORMAT` | `output.format` |
| `NGINX_LENS_AGENT_TOKEN` | `web.agent.token` |
| `NGINX_LENS_HUB_TOKEN` | `web.hub.token` |

## Секции config.yaml

### `defaults`

| Поле | Описание | По умолчанию |
|------|----------|--------------|
| `nginx_config_path` | Путь к nginx.conf (**обязательно**) | `/etc/nginx/nginx.conf` |
| `timeout` | Таймаут health-check (сек) | `2.0` |
| `retries` | Повторы health-check | `1` |
| `mode` | `tcp` или `http` | `tcp` |
| `max_workers` | Параллелизм DNS/health | `10` |
| `dns_cache_ttl` | Устаревшее; используйте `cache.ttl` | `300` |
| `top` | Top-N в logs | `10` |

### `parser`

| Поле | Значения |
|------|----------|
| `mode` | `auto` — expanded если nginx в PATH, иначе regex; `expanded`; `regex` |
| `nginx_path` | Бинарник nginx для `-T` / `-t` |

### `docker`

| Поле | Описание |
|------|----------|
| `enabled` | `auto` \| `true` \| `false` |
| `container` | Имя контейнера (пусто = auto-detect `*nginx*`) |
| `binary` | `docker` |
| `config_inside` | Путь nginx.conf внутри контейнера |
| `volume_map` | `{путь_в_контейнере: путь_на_хосте}` |

### `output`

| Поле | Значения |
|------|----------|
| `format` | `table`, `json`, `yaml`, `csv`, `prometheus` |
| `colors` | ANSI-цвета (отключается `NO_COLOR=1`) |

### `cache`

DNS-кэш для `resolve` / `health`:

| Поле | Описание |
|------|----------|
| `enabled` | Включить кэш |
| `ttl` | TTL секунд |

Per-command: `health.skip_cache`, `resolve.skip_cache`.

### `validate`

CI-пайплайн `nginx-lens validate`. `skip_*: true` — пропустить шаг.

| Поле | Описание |
|------|----------|
| `skip_syntax` | Пропустить `nginx -t` |
| `skip_analysis` | Пропустить статический анализ |
| `skip_upstream` | Пропустить health upstream |
| `skip_dns` | Пропустить DNS resolve |
| `skip_certs` | Пропустить аудит сертификатов |
| `skip_policy` | Пропустить policy engine |
| `skip_warns` | Игнорировать `[warn]` в `nginx -t` |
| `fail_on` | `low` \| `medium` \| `high` — минимальная severity для exit 1 |
| `fail_on_low` | Устарело; используйте `fail_on: low` |
| `nginx_path` | Бинарник для syntax-check |

`policy.policy_only: true` — только policy, без syntax/upstream.

### `analyze`

| Поле | Описание |
|------|----------|
| `min_severity` | `low`, `medium`, `high` |
| `skip_types` | Список типов issues для пропуска |
| `skip_low` / `skip_medium` | Фильтр severity |

### `syntax`

| Поле | Описание |
|------|----------|
| `skip_warns` | Не падать на warn в `nginx -t` |
| `nginx_path` | Бинарник nginx |

### `health` / `resolve`

| Секция | Поле | Описание |
|--------|------|----------|
| `health` | `with_resolve` | Показать resolved IP |
| `health` | `skip_exit_on_unhealthy` | Не exit 1 при down upstream |
| `health` | `skip_cache` | Без DNS-кэша |
| `resolve` | `skip_cache` | Без DNS-кэша |

### `route` / `explain` / `blast_radius`

| Секция | Поле | CLI fallback |
|--------|------|--------------|
| `route` | `url` | `nginx-lens route [url]` или `--url` |
| `explain` | `url` | `nginx-lens explain [url]` |
| `blast_radius` | `upstream_name` | `nginx-lens blast-radius [name]` |

### `policy`

| Поле | Описание |
|------|----------|
| `packs` | Встроенные наборы правил (см. ниже) |
| `rules` | Custom rules (`id`, `match`, `severity`, `message`) |
| `policy_only` | Только policy в `validate` |

**Встроенные packs:** `security-baseline`, `mozilla-ssl`, `performance`, `performance-baseline`, `caching`, `rate-limit`, `custom-*`.

Пример custom rule:

```yaml
policy:
  rules:
    - id: no-root
      match: "directive.root /var/www"
      severity: medium
      message: "root в location без alias"
```

### `certs`

| Поле | Описание |
|------|----------|
| `warn_days` | Предупреждение до истечения (дней) |
| `fail_on_expired` | Exit 1 при просроченном cert |

### `score`

| Поле | Описание |
|------|----------|
| `enabled` | Учитывать score в agent snapshot |

### `k8s`

| Поле | Описание |
|------|----------|
| `manifests_path` | Директория YAML для `ingress-audit` |

### `logs`

| Поле | Описание |
|------|----------|
| `path` | Access log |
| `error_path` | Error log (агент → Hub correlation) |
| `top` | Top-N путей/IP |
| `since` / `until` | Фильтр по времени |
| `status` | Фильтр HTTP status |
| `skip_anomalies` | `false` — включить детект аномалий |
| `tail_lines` | Макс. строк для чтения |

### `diff`

| Поле | CLI fallback |
|------|--------------|
| `config1` | 1-й аргумент / `--config1` |
| `config2` | 2-й аргумент / `--config2` |

### `metrics`

| Поле | Описание |
|------|----------|
| `compare_path` | Второй конфиг для сравнения метрик |
| `prometheus` | Вывод Prometheus text format |

### `tree` / `include_tree`

| Секция | Поле | Значения |
|--------|------|----------|
| `tree` | `format` | `text`, `markdown`, `html` |
| `include_tree` | `directive` | Фильтр shadowing по директиве |

### `upstreams`

| Поле | Описание |
|------|----------|
| `name` | Фильтр по имени upstream |
| `health` | Health-check в выводе |
| `health_by_default` | Health для всех блоков |
| `skip_health` | Отключить health |

### `dynamic_upstream`

Интеграция с [ngx_dynamic_upstream](https://github.com/ZigzagAK/ngx_dynamic_upstream):

```yaml
dynamic_upstream:
  enabled: true
  api_url: http://127.0.0.1:6000/dynamic
  timeout: 2.0
```

| Сценарий | Поведение |
|----------|-----------|
| Upstream с `server` в конфиге | Работает как обычно |
| Только `dynamic_state_file`, без `server` | Пустой список, если API выключен |
| `enabled: true` + доступный API | Серверы подтягиваются из HTTP API модуля |

Пример nginx location для API:

```nginx
location /dynamic {
    allow 127.0.0.1;
    deny all;
    dynamic_upstream;
}
```

### `web`

См. [WEB_DEPLOYMENT.md](WEB_DEPLOYMENT.md).

```yaml
web:
  agent:
    host: 0.0.0.0
    port: 8088
    token: ""
  hub:
    host: 0.0.0.0
    port: 8089
    token: ""
    agent_token: ""
    agents:
      - http://10.0.0.5:8088
    cors_origins:
      - "*"
    refresh_interval: 30
```

## Agent Prometheus

`GET /metrics` на агенте (без auth):

- `nginx_lens_score`, `nginx_lens_issues_high`
- `nginx_lens_upstream_healthy{upstream,address}`
- `nginx_lens_cert_days_left_min`, `nginx_lens_access_rps`

Полные HTTP-схемы — [API.md](API.md).
