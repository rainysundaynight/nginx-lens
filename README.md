# nginx-lens

CLI-инструмент для анализа, визуализации и диагностики конфигураций Nginx.

Все настройки — в `config.yaml`. Команды вызываются без аргументов.

## Установка

### С GitHub Releases (рекомендуется)

**Linux / macOS** — один скрипт, три бинарника в PATH:

```bash
curl -fsSL https://raw.githubusercontent.com/rainysundaynight/nginx-lens/main/install-nginx-lens.sh | bash
```

Конкретная версия:

```bash
NGINX_LENS_VERSION=2.3.0 curl -fsSL https://raw.githubusercontent.com/rainysundaynight/nginx-lens/main/install-nginx-lens.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/rainysundaynight/nginx-lens/main/install.ps1 | iex
```

**Вручную** — скачайте архив с [GitHub Releases](https://github.com/rainysundaynight/nginx-lens/releases):

| Платформа | Архив |
|-----------|-------|
| Linux amd64 | `nginx-lens_<version>_Linux_amd64.tar.gz` |
| Linux arm64 | `nginx-lens_<version>_Linux_arm64.tar.gz` |
| macOS amd64 | `nginx-lens_<version>_Darwin_amd64.tar.gz` |
| macOS arm64 | `nginx-lens_<version>_Darwin_arm64.tar.gz` |
| Windows amd64 | `nginx-lens_<version>_Windows_amd64.zip` |

В архиве: `nginx-lens`, `nginx-lens-agent`, `nginx-lens-hub`, `example-config.yaml`.

```bash
tar xzf nginx-lens_2.3.0_Linux_amd64.tar.gz
sudo install nginx-lens nginx-lens-agent nginx-lens-hub /usr/local/bin/
sudo nginx-lens init
```

### Из исходников

```bash
git clone https://github.com/rainysundaynight/nginx-lens.git
cd nginx-lens && make build
sudo make install
```

## Первичная настройка

```bash
sudo nginx-lens init          # /opt/nginx-lens/config.yaml + shell completion
# или: nginx-lens init --user  # ~/.nginx-lens/config.yaml
```

Обязательно укажите `defaults.nginx_config_path` — путь к вашему `nginx.conf`.

### Nginx в Docker

```yaml
docker:
  enabled: auto
  container: nginx
  config_inside: /etc/nginx/nginx.conf
  volume_map:
    /etc/nginx: /opt/nginx/conf
    /var/log/nginx: /var/log/nginx
```

Режим `auto`: если конфиг доступен на хосте через `volume_map` — читает файл; иначе `docker exec nginx -T`.

Пример полного конфига: [example-config.yaml](example-config.yaml)

## Быстрый старт

```bash
# Все команды читают настройки из config.yaml
nginx-lens health
nginx-lens analyze
nginx-lens validate
nginx-lens route http://example.com/api/v1/users
nginx-lens explain --url http://example.com/api/v1/users
nginx-lens blast-radius api_backend
nginx-lens diff old/nginx.conf new/nginx.conf
nginx-lens certs      # SSL/TLS аудит сертификатов
nginx-lens score      # рейтинг конфигурации 0-100
nginx-lens ingress-audit  # k8s.manifests_path
nginx-lens config validate  # проверка схемы config.yaml
nginx-lens logs       # logs.path в конфиге
nginx-lens metrics
nginx-lens tree
nginx-lens syntax
```

### Пример config.yaml

```yaml
defaults:
  nginx_config_path: /etc/nginx/nginx.conf
  timeout: 3.0
  mode: tcp

output:
  format: table  # table | json | yaml

route:
  url: http://example.com/api/v1/users

logs:
  path: /var/log/nginx/access.log
  top: 20

validate:
  skip_dns: true
```

JSON-вывод для CI:

```yaml
output:
  format: json
```

```bash
nginx-lens validate
```

## Web-стек

Настройки в секции `web:` конфига.

```bash
nginx-lens-agent   # web.agent.host / web.agent.port
nginx-lens-hub     # web.hub.host / web.hub.port
```

## Команды

| Команда | Секция конфига |
|---------|----------------|
| `health` | `defaults`, `health`, `cache`, `output` |
| `resolve` | `defaults`, `resolve`, `cache`, `output` |
| `analyze` | `defaults`, `analyze`, `output` |
| `validate` | `validate`, `defaults`, `output` |
| `syntax` | `syntax`, `defaults` |
| `route` | `--url` / аргумент; fallback: `route.url` |
| `explain` | `--url` / аргумент; fallback: `explain.url`, `route.url` |
| `blast-radius` | `--upstream` / аргумент; fallback: `blast_radius.upstream_name` |
| `certs` | `certs` |
| `score` | `score` |
| `ingress-audit` | `k8s.manifests_path` |
| `config validate` | — |
| `tree` | `tree.format`, `defaults` |
| `graph` | `defaults` |
| `include-tree` | `include_tree`, `defaults` |
| `diff` | `--config1`, `--config2` / аргументы; fallback: `diff.config1`, `diff.config2` |
| `logs` | `logs`, `output` |
| `metrics` | `metrics`, `defaults`, `output` |
| `upstreams` | `upstreams`, `defaults`, `output` |
| `config` | — |
| `init` | — |

## Поиск конфига

1. `NGINX_LENS_CONFIG` env
2. `/opt/nginx-lens/config.yaml`
3. `.nginx-lens.yaml` в текущей директории
4. `~/.nginx-lens/config.yaml`

## Ограничения

| Режим / функция | Когда использовать | Ограничения |
|-----------------|-------------------|-------------|
| `parser.mode: expanded` | nginx установлен и доступен в PATH | Требует успешный `nginx -t`; конфиг должен быть валиден для nginx |
| `parser.mode: regex` | CI/CD без nginx, статический разбор файлов | Не понимает `map`, сложный `if`, макросы; dynamic upstream API не учитывается |
| `route` / `explain` | Статический анализ маршрутизации | Без runtime-переменных (`$arg_*`, `$cookie_*`); rewrite trace — декларативный |
| `ingress-audit` | Сверка K8s Ingress с nginx | Walk по YAML-манифестам, не полная семантика ingress-nginx controller |
| `policy` packs | CI baseline-проверки | Встроенные правила; для custom — `policy.rules` или pack `custom-*` |

## Документация

Полный индекс: **[docs/README.md](docs/README.md)**

| Документ | Описание |
|----------|----------|
| [Руководство](docs/guide.md) | Установка, сценарии, checklist |
| [Конфигурация](docs/CONFIGURATION.md) | Все секции config.yaml |
| [API](docs/API.md) | JSON-схемы и HTTP agent/hub |
| [Web-стек](docs/WEB_DEPLOYMENT.md) | Agent, Hub, dashboard |

## Production checklist

Перед выкладкой на prod-хост:

1. `nginx-lens init` и **`nginx-lens config validate`** (схема config.yaml, policy packs)
2. `parser.mode: auto` (или `expanded` если nginx в PATH)
3. `defaults.nginx_config_path` — реальный `nginx.conf`
4. `logs.error_path` на агенте — для корреляции error.log в Hub
5. `policy.packs` — включить `security-baseline` и `mozilla-ssl` для CI
6. `nginx-lens validate` в pipeline с `output.format: json`
7. Agent token (`web.agent.token`) и Hub token (`web.hub.token`) — не пустые в prod

```bash
make build
make test-coverage
nginx-lens config validate
nginx-lens validate
nginx-lens score
```

## Разработка

```bash
make test
make lint
make build
```
