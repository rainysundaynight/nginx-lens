# Конфигурация nginx-lens

`nginx-lens` может быть настроен через конфигурационный файл, переменные окружения или параметры командной строки (в порядке убывания приоритета: CLI опции > переменные окружения > конфигурационный файл).

## Конфигурационный файл

Конфигурационный файл может быть размещен в следующих местах (в порядке приоритета):

1. `.nginx-lens.yaml` или `.nginx-lens.yml` в текущей директории
2. `/opt/nginx-lens/config.yaml` или `/opt/nginx-lens/config.yml` (системный конфиг)
3. `~/.nginx-lens/config.yaml` или `~/.nginx-lens/config.yml` (пользовательский конфиг)

### Пример конфигурационного файла

```yaml
defaults:
  timeout: 2.0          # Таймаут проверки upstream (сек)
  retries: 1            # Количество попыток
  mode: tcp             # Режим проверки: tcp или http
  max_workers: 10       # Максимальное количество потоков для параллельной обработки
  dns_cache_ttl: 300   # Время жизни DNS кэша (сек)
  top: 10               # Количество топ-значений для команды logs
  nginx_config_path: /etc/nginx/nginx.conf  # Путь к nginx.conf (по умолчанию /etc/nginx/nginx.conf, если файл не существует - используется автопоиск)

output:
  colors: true          # Использовать цвета в выводе
  format: table         # Формат вывода: table, json, yaml

cache:
  enabled: true         # Включить DNS кэширование
  ttl: 300              # Время жизни кэша (сек)

validate:
  check_syntax: true    # Проверять синтаксис через nginx -t
  check_analysis: true  # Выполнять анализ проблем
  check_upstream: true  # Проверять доступность upstream
  check_dns: false      # Проверять DNS резолвинг upstream (по умолчанию выключено)
  nginx_path: nginx     # Путь к бинарю nginx

dynamic_upstream:
  enabled: false        # Включить интеграцию с ngx_dynamic_upstream (по умолчанию выключено)
  api_url: http://127.0.0.1:6000/dynamic  # URL endpoint модуля ngx_dynamic_upstream
  timeout: 2.0          # Таймаут HTTP запросов к API (сек)
```

## Переменные окружения

Все параметры могут быть заданы через переменные окружения, используя префикс `NGINX_LENS_` и заменяя дефисы на подчеркивания:

- `NGINX_LENS_TIMEOUT` → `--timeout`
- `NGINX_LENS_MAX_WORKERS` → `--max-workers`
- `NGINX_LENS_NO_CACHE` → `--no-cache`

## Команды и опции

### Общие опции

Многие команды поддерживают следующие общие опции:

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |
| `--max-workers`, `-w` | int | Максимальное количество потоков для параллельной обработки | `10` (из конфига) |
| `--no-cache` | bool | Отключить кэширование DNS резолвинга | `false` |
| `--cache-ttl` | int | Время жизни кэша в секундах | `300` (из конфига) |

---

## Команды

### `health`

Проверяет доступность upstream-серверов, определённых в nginx.conf. Выводит таблицу со статусом каждого сервера.

**Синтаксис:**
```bash
nginx-lens health [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига (`defaults.nginx_config_path`) или автопоиск в стандартных местах

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--timeout` | float | Таймаут проверки (сек) | `2.0` (из конфига) |
| `--retries` | int | Количество попыток | `1` (из конфига) |
| `--mode` | string | Режим проверки: `tcp` или `http` | `tcp` (из конфига) |
| `--resolve`, `-r` | bool | Показать резолвленные IP-адреса | `false` |
| `--max-workers`, `-w` | int | Максимальное количество потоков | `10` (из конфига) |
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |
| `--no-cache` | bool | Отключить кэширование DNS резолвинга | `false` |
| `--cache-ttl` | int | Время жизни кэша в секундах | `300` (из конфига) |

**Примеры:**
```bash
# Базовая проверка (использует путь из конфига)
nginx-lens health

# С указанием пути
nginx-lens health /etc/nginx/nginx.conf

# С дополнительными опциями
nginx-lens health /etc/nginx/nginx.conf --timeout 5 --retries 3 --mode http

# С резолвингом DNS (использует путь из конфига)
nginx-lens health --resolve

# Экспорт в JSON
nginx-lens health --resolve --json

# С увеличенным количеством потоков
nginx-lens health --max-workers 20
```

**Exit codes:**
- `0` - Все upstream серверы доступны
- `1` - Обнаружены недоступные серверы или проблемы с DNS

---

### `resolve`

Резолвит DNS имена upstream-серверов в IP-адреса. Показывает все IP-адреса для каждого hostname, включая обработку CNAME записей.

**Синтаксис:**
```bash
nginx-lens resolve [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--max-workers`, `-w` | int | Максимальное количество потоков | `10` (из конфига) |
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |
| `--no-cache` | bool | Отключить кэширование DNS резолвинга | `false` |
| `--cache-ttl` | int | Время жизни кэша в секундах | `300` (из конфига) |

**Примеры:**
```bash
# Базовый резолвинг (использует путь из конфига)
nginx-lens resolve

# С указанием пути
nginx-lens resolve /etc/nginx/nginx.conf

# С увеличенным количеством потоков
nginx-lens resolve --max-workers 20

# Без кэширования
nginx-lens resolve --no-cache

# Экспорт в YAML
nginx-lens resolve --yaml
```

**Exit codes:**
- `0` - Все DNS имена резолвятся корректно
- `1` - Обнаружены проблемы с DNS резолвингом (невалидные CNAME, нерезолвящиеся имена)

---

### `analyze`

Анализирует конфигурацию Nginx на типовые проблемы и best practices.

**Синтаксис:**
```bash
nginx-lens analyze [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |

**Проверяемые проблемы:**

| Тип проблемы | Критичность | Описание |
|--------------|-------------|----------|
| `location_conflict` | medium | Возможное пересечение location |
| `duplicate_directive` | medium | Дублирующиеся директивы |
| `empty_block` | low | Пустые блоки |
| `proxy_pass_no_scheme` | medium | proxy_pass без схемы |
| `autoindex_on` | medium | autoindex включен |
| `if_block` | medium | if внутри location |
| `server_tokens_on` | low | server_tokens включен |
| `ssl_missing` | high | Отсутствует SSL сертификат/ключ |
| `ssl_protocols_weak` | high | Устаревшие TLS протоколы |
| `ssl_ciphers_weak` | high | Слабые шифры |
| `listen_443_no_ssl` | high | listen 443 без ssl |
| `listen_443_no_http2` | low | listen 443 без http2 |
| `no_limit_req_conn` | medium | Отсутствуют limit_req/limit_conn |
| `missing_security_header` | medium | Отсутствуют security заголовки |
| `deprecated` | medium | Устаревшие директивы |
| `limit_too_small` | medium | Слишком маленькие лимиты |
| `limit_too_large` | medium | Слишком большие лимиты |
| `unused_variable` | low | Неиспользуемые переменные |
| `listen_servername_conflict` | high | Конфликты listen/server_name |
| `rewrite_cycle` | high | Циклические rewrite правила |
| `rewrite_conflict` | medium | Конфликты rewrite |
| `rewrite_no_flag` | low | rewrite без флага |
| `dead_location` | low | Мертвые location |

**Примеры:**
```bash
# Базовый анализ (использует путь из конфига)
nginx-lens analyze

# С указанием пути
nginx-lens analyze /etc/nginx/nginx.conf

# Экспорт в JSON
nginx-lens analyze --json
```

**Exit codes:**
- `0` - Проблем не найдено
- `1` - Обнаружены проблемы

---

### `validate`

Комплексная валидация конфигурации Nginx. Выполняет проверку синтаксиса, анализ проблем, проверку доступности upstream и опционально DNS резолвинг.

**Синтаксис:**
```bash
nginx-lens validate [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--nginx-path` | string | Путь к бинарю nginx | `nginx` (из конфига) |
| `--syntax` / `--no-syntax` | bool | Проверять синтаксис через nginx -t | `true` (из конфига) |
| `--analysis` / `--no-analysis` | bool | Выполнять анализ проблем | `true` (из конфига) |
| `--upstream` / `--no-upstream` | bool | Проверять доступность upstream | `true` (из конфига) |
| `--dns` / `--no-dns` | bool | Проверять DNS резолвинг upstream | `false` (из конфига) |
| `--timeout` | float | Таймаут проверки upstream (сек) | `2.0` (из конфига) |
| `--max-workers`, `-w` | int | Максимальное количество потоков | `10` (из конфига) |
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |

**Примеры:**
```bash
# Полная валидация (использует путь из конфига)
nginx-lens validate

# С указанием пути
nginx-lens validate /etc/nginx/nginx.conf

# Без проверки upstream
nginx-lens validate --no-upstream

# С DNS проверкой и экспортом в JSON
nginx-lens validate --dns --json

# Только синтаксис
nginx-lens validate --no-analysis --no-upstream --no-dns
```

**Exit codes:**
- `0` - Валидация пройдена успешно
- `1` - Обнаружены проблемы

---

### `syntax`

Проверяет синтаксис nginx-конфига через `nginx -t` с подсветкой ошибок.

**Синтаксис:**
```bash
nginx-lens syntax [OPTIONS]
```

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `-c`, `--config` | string | Путь к кастомному nginx.conf | Автопоиск |
| `--nginx-path` | string | Путь к бинарю nginx | `nginx` |

**Автопоиск конфига:**
Если `-c` не указан, ищет конфиг в следующем порядке:
1. `/etc/nginx/nginx.conf`
2. `/usr/local/etc/nginx/nginx.conf`
3. `./nginx.conf`

**Примеры:**
```bash
# Автопоиск конфига
nginx-lens syntax

# Указать путь к конфигу
nginx-lens syntax -c /etc/nginx/nginx.conf

# Указать путь к бинарю nginx
nginx-lens syntax --nginx-path /usr/local/bin/nginx
```

---

### `tree`

Показывает древовидную структуру конфигурации Nginx.

**Синтаксис:**
```bash
nginx-lens tree [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--markdown` | bool | Экспортировать в Markdown | `false` |
| `--html` | bool | Экспортировать в HTML | `false` |

**Примеры:**
```bash
# Базовый вывод (использует путь из конфига)
nginx-lens tree

# С указанием пути
nginx-lens tree /etc/nginx/nginx.conf

# Экспорт в Markdown
nginx-lens tree --markdown > config-tree.md

# Экспорт в HTML
nginx-lens tree --html > config-tree.html
```

---

### `diff`

Сравнивает два конфигурационных файла и показывает различия.

**Синтаксис:**
```bash
nginx-lens diff <config1> <config2>
```

**Аргументы:**
- `config1` (обязательный) - Путь к первому nginx.conf
- `config2` (обязательный) - Путь ко второму nginx.conf

**Примеры:**
```bash
# Сравнение двух конфигов
nginx-lens diff /etc/nginx/nginx.conf /etc/nginx/nginx.conf.backup
```

**Exit codes:**
- `0` - Конфиги идентичны
- `1` - Обнаружены различия или ошибки

---

### `route`

Показывает, какой server/location обслуживает указанный URL.

**Синтаксис:**
```bash
nginx-lens route <url> [OPTIONS]
```

**Аргументы:**
- `url` (обязательный) - URL для маршрутизации (например, `http://host/path`)

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `-c`, `--config` | string | Путь к кастомному nginx.conf | Поиск в `/etc/nginx/**/*.conf` |

**Примеры:**
```bash
# Поиск маршрута (автопоиск конфигов в /etc/nginx)
nginx-lens route http://example.com/api/v1

# С указанием конфига
nginx-lens route -c /etc/nginx/nginx.conf http://example.com/api/v1
```

---

### `include-tree`

Показывает дерево include-ов, циклы и shadowing директив.

**Синтаксис:**
```bash
nginx-lens include-tree [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--directive` | string | Показать shadowing для директивы (например, `server_name`) | `None` |

**Примеры:**
```bash
# Базовый вывод дерева include (использует путь из конфига)
nginx-lens include-tree

# С указанием пути
nginx-lens include-tree /etc/nginx/nginx.conf

# Показать shadowing для server_name
nginx-lens include-tree --directive server_name
```

---

### `graph`

Визуализирует маршруты и структуру конфигурации в виде графа.

**Синтаксис:**
```bash
nginx-lens graph [config_path]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Примеры:**
```bash
# Визуализация графа (использует путь из конфига)
nginx-lens graph

# С указанием пути
nginx-lens graph /etc/nginx/nginx.conf
```

---

### `logs`

Анализирует access.log/error.log. Показывает топ-статусы, пути, IP, User-Agent, ошибки, анализ времени ответа и обнаружение аномалий.

**Синтаксис:**
```bash
nginx-lens logs <log_path> [OPTIONS]
```

**Аргументы:**
- `log_path` (обязательный) - Путь к access.log или error.log (поддерживается gzip)

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--top` | int | Сколько топ-значений выводить | `10` (из конфига) |
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |
| `--csv` | bool | Экспортировать результаты в CSV | `false` |
| `--since` | string | Фильтр: с даты (формат: `YYYY-MM-DD` или `YYYY-MM-DD HH:MM:SS`) | `None` |
| `--until` | string | Фильтр: до даты (формат: `YYYY-MM-DD` или `YYYY-MM-DD HH:MM:SS`) | `None` |
| `--status` | string | Фильтр по статусам (например: `404,500`) | `None` |
| `--detect-anomalies` | bool | Обнаруживать аномалии в логах | `false` |

**Примеры:**
```bash
# Базовый анализ
nginx-lens logs /var/log/nginx/access.log

# С фильтрацией по времени
nginx-lens logs /var/log/nginx/access.log --since "2024-01-01" --until "2024-01-31"

# Фильтр по статусам ошибок
nginx-lens logs /var/log/nginx/access.log --status 404,500

# С обнаружением аномалий
nginx-lens logs /var/log/nginx/access.log --detect-anomalies

# Анализ gzip архива
nginx-lens logs /var/log/nginx/access.log.gz --top 20

# Экспорт в CSV
nginx-lens logs /var/log/nginx/access.log --csv > analysis.csv
```

---

### `metrics`

Собирает метрики о конфигурации Nginx. Показывает количество блоков, директив, статистику по server блокам и поддерживает сравнение метрик между версиями.

**Синтаксис:**
```bash
nginx-lens metrics [config_path] [OPTIONS]
```

**Аргументы:**
- `config_path` (опциональный) - Путь к nginx.conf. Если не указан, используется путь из конфига или автопоиск

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--compare`, `-c` | string | Путь к другому конфигу для сравнения | `None` |
| `--prometheus`, `-p` | bool | Экспортировать в формате Prometheus | `false` |
| `--json` | bool | Экспортировать результаты в JSON | `false` |
| `--yaml` | bool | Экспортировать результаты в YAML | `false` |

**Примеры:**
```bash
# Базовый сбор метрик (использует путь из конфига)
nginx-lens metrics

# С указанием пути
nginx-lens metrics /etc/nginx/nginx.conf

# Экспорт в Prometheus
nginx-lens metrics --prometheus

# Сравнение с предыдущей версией
nginx-lens metrics --compare /etc/nginx/nginx.conf.old

# Экспорт в JSON
nginx-lens metrics --json
```

---

### `init`

Инициализирует nginx-lens: создает конфиг и устанавливает автодополнение.

**Синтаксис:**
```bash
nginx-lens init [OPTIONS]
```

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--force`, `-f` | bool | Перезаписать существующий конфиг | `false` |

**Примеры:**
```bash
# Базовая инициализация
nginx-lens init

# С перезаписью конфига
nginx-lens init --force

# С правами root (для создания /opt/nginx-lens)
sudo nginx-lens init
```

---

### `completion`

Генерация скриптов автодополнения для shell.

#### `completion install`

Генерирует скрипт автодополнения для указанного shell.

**Синтаксис:**
```bash
nginx-lens completion install <shell> [OPTIONS]
```

**Аргументы:**
- `shell` (обязательный) - Тип shell: `bash`, `zsh`, `fish`, `powershell`

**Опции:**

| Опция | Тип | Описание | По умолчанию |
|-------|-----|----------|--------------|
| `--output`, `-o` | string | Путь для сохранения скрипта | stdout |

**Примеры:**
```bash
# Bash
nginx-lens completion install bash >> ~/.bashrc

# Zsh
nginx-lens completion install zsh >> ~/.zshrc

# Fish
nginx-lens completion install fish > ~/.config/fish/completions/nginx-lens.fish

# PowerShell
nginx-lens completion install powershell > nginx-lens-completion.ps1
```

#### `completion show-instructions`

Показывает инструкции по установке автодополнения для всех поддерживаемых shell.

**Синтаксис:**
```bash
nginx-lens completion show-instructions
```

---

## Exit Codes

Все команды возвращают стандартные exit codes для интеграции с CI/CD и скриптами:

- `0` - Успешное выполнение
- `1` - Обнаружены проблемы или ошибки
- `130` - Операция отменена пользователем (Ctrl+C)

---

## DNS Кэширование

nginx-lens использует файловый кэш для DNS резолвинга, чтобы ускорить повторные запуски. Кэш хранится в `~/.nginx-lens/dns_cache.json` и имеет настраиваемый TTL (по умолчанию 300 секунд).

**Управление кэшем:**

- `--no-cache` - Отключить кэширование для текущего запуска
- `--cache-ttl <seconds>` - Установить время жизни кэша
- Конфигурация через `cache.enabled` и `cache.ttl` в конфигурационном файле

---

## Интеграция с ngx_dynamic_upstream

Начиная с версии 0.6.0, `nginx-lens` поддерживает интеграцию с модулем [`ngx_dynamic_upstream`](https://github.com/ZigzagAK/ngx_dynamic_upstream) для работы с динамическими upstream блоками.

**Настройка:**

```yaml
dynamic_upstream:
  enabled: true                    # Включить интеграцию
  api_url: http://127.0.0.1:6000/dynamic  # URL endpoint модуля
  timeout: 2.0                     # Таймаут HTTP запросов (сек)
```

**Как это работает:**

1. Парсер находит все upstream блоки в конфиге
2. Для upstream блоков без серверов (только с `dynamic_state_file`) выполняется HTTP запрос к API модуля
3. Полученные серверы добавляются к списку upstream
4. Команды `health`, `resolve` и `validate` работают с обогащенными данными

**Пример конфигурации nginx:**

```nginx
upstream backends2 {
    zone zone_for_backends2 1m;
    dynamic_state_file backend.peers;
}

server {
    listen 6000;
    
    location /dynamic {
        allow 127.0.0.1;
        deny all;
        dynamic_upstream;
    }
}
```

**Примечания:**

- Если API недоступен или возвращает ошибку, команды продолжают работу с серверами из конфига
- Динамические upstream без серверов в конфиге пропускаются, если интеграция отключена или API недоступен
- Ошибки подключения к API логируются, но не прерывают выполнение команд

Подробнее см. [DYNAMIC_UPSTREAM_COMPATIBILITY.md](../docs/DYNAMIC_UPSTREAM_COMPATIBILITY.md)

---

## Прогресс-бары

Для операций с большим количеством задач (более 5) автоматически отображаются прогресс-бары с ETA. Операцию можно отменить через Ctrl+C.

---

## Примеры использования

### CI/CD интеграция

```bash
# Валидация конфига перед деплоем
nginx-lens validate /etc/nginx/nginx.conf --json > validation.json
if [ $? -ne 0 ]; then
    echo "Валидация не пройдена"
    exit 1
fi
```

### Мониторинг метрик

```bash
# Экспорт метрик в Prometheus
nginx-lens metrics /etc/nginx/nginx.conf --prometheus | \
    curl -X POST http://prometheus:9091/metrics/job/nginx-config --data-binary @-
```

### Анализ логов

```bash
# Анализ логов за последний день
nginx-lens logs /var/log/nginx/access.log \
    --since "$(date -d '1 day ago' +%Y-%m-%d)" \
    --detect-anomalies \
    --json > daily-analysis.json
```

### Массовая проверка upstream

```bash
# Проверка всех upstream с резолвингом DNS
nginx-lens health /etc/nginx/nginx.conf \
    --resolve \
    --max-workers 50 \
    --timeout 5 \
    --json > health-check.json
```

---

## Дополнительная информация

Для получения справки по любой команде используйте:
```bash
nginx-lens <command> --help
```

Для получения списка всех доступных команд:
```bash
nginx-lens --help
```

