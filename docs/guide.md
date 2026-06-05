# Руководство

## Установка

### Releases (рекомендуется)

```bash
curl -fsSL https://raw.githubusercontent.com/rainysundaynight/nginx-lens/main/install-nginx-lens.sh | bash
```

В архиве: `nginx-lens`, `nginx-lens-agent`, `nginx-lens-hub`, `example-config.yaml`.

### Из исходников

```bash
git clone https://github.com/rainysundaynight/nginx-lens.git
cd nginx-lens && make build && sudo make install
```

## Первый запуск

```bash
sudo nginx-lens init
# отредактируйте /opt/nginx-lens/config.yaml — минимум defaults.nginx_config_path

export NGINX_LENS_CONFIG=/opt/nginx-lens/config.yaml   # опционально
nginx-lens config validate
nginx-lens analyze
nginx-lens health
```

Пользовательский конфиг без root:

```bash
nginx-lens init --user    # ~/.nginx-lens/config.yaml
```

## Типовые сценарии

### CI/CD

```yaml
# config.yaml на runner
output:
  format: json
validate:
  skip_dns: true
  fail_on: high
policy:
  packs:
    - security-baseline
    - mozilla-ssl
```

```bash
nginx-lens config validate
nginx-lens validate
# exit 1 при high issues, syntax/upstream/dns failures
```

### Маршрутизация URL

```bash
nginx-lens route --url http://example.com/api/v1/users
nginx-lens explain --url http://example.com/api/v1/users
```

Или задайте `route.url` / `explain.url` в config.yaml и вызывайте команды без аргументов.

### Сравнение конфигов

```bash
nginx-lens diff /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak
```

Или `diff.config1` / `diff.config2` в config.yaml.

### Nginx в Docker

```yaml
docker:
  enabled: auto
  volume_map:
    /etc/nginx: /host/nginx/conf
    /var/log/nginx: /var/log/nginx
defaults:
  nginx_config_path: /etc/nginx/nginx.conf
```

Режим `auto`: читает файл с хоста через `volume_map`; иначе — `docker exec … nginx -T`.

### Policy и score

```yaml
policy:
  packs:
    - security-baseline
    - mozilla-ssl
    - performance-baseline
score:
  enabled: true
```

```bash
nginx-lens score
nginx-lens validate   # policy включён в pipeline, если skip_policy: false
```

## Команды

| Команда | Назначение |
|---------|------------|
| `health` | TCP/HTTP проверка upstream |
| `resolve` | DNS upstream → IP |
| `analyze` | Статический анализ (conflicts, SSL, headers, …) |
| `validate` | Полный CI-пайплайн |
| `syntax` | `nginx -t` |
| `route` / `explain` | Маршрут URL → server/location |
| `blast-radius` | Impact при падении upstream |
| `certs` | Аудит SSL/TLS |
| `score` | Рейтинг 0–100 |
| `diff` | Семантическое сравнение конфигов |
| `ingress-audit` | K8s Ingress vs `server_name` |
| `logs` | Статистика access.log |
| `metrics` | Счётчики server/location/upstream |
| `tree` / `graph` / `include-tree` | Визуализация |
| `upstreams` | Сводка upstream-блоков |
| `config validate` | Схема config.yaml |

Секции конфига для каждой команды — в [CONFIGURATION.md](CONFIGURATION.md).

## Production checklist

1. `nginx-lens init` → **`nginx-lens config validate`**
2. `defaults.nginx_config_path` — реальный `nginx.conf`
3. `parser.mode: auto` (или `expanded` на хосте с nginx)
4. `policy.packs`: `security-baseline`, `mozilla-ssl` для CI
5. `nginx-lens validate` с `output.format: json`
6. Agent/Hub: непустые `web.agent.token`, `web.hub.token`
7. На агенте: `logs.error_path` для корреляции error.log в Hub

## Ограничения

- **Config-first:** параметры задаются в `config.yaml`, а не через CLI-флаги.
- **route/explain:** статический анализ, без runtime `$variables`.
- **ingress-audit:** walk по YAML, не полная семантика ingress-nginx controller.
- **regex-парсер:** упрощённый; для сложных конфигов используйте `expanded` / `auto`.

## Разработка

```bash
make test
make test-coverage
make lint
make build
```

История версий: [CHANGELOG.md](../CHANGELOG.md).
