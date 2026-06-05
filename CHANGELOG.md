# Changelog

## [1.1.4] — 2026-06-05

### Fixed

- Correlations: 5xx из access.log без `$upstream_addr` теперь привязываются к upstream по URI через dependency graph (path → location)
- Резолв upstream из error.log для адресов вида `http://127.0.0.1:80/`
- Hub ERRORS: показывается логическое имя upstream и текст ошибки, а не сырой URL
- CLI logs: P95 latency помечается как n/a, если в log_format нет `$request_time`

## [1.1.3] — 2026-06-05

### Fixed

- Дубликаты issues в Hub при парсинге `nginx -T`: expanded-секции уже содержат include-файлы, повторный разбор `include` больше не дублирует server-блоки
- Дедупликация issues по `(type, file, line, description)` в `CollectIssues` и `AppendIssue`

## [1.1.2] — 2026-06-05

### Added

- Предупреждение `server_tokens_off_missing` (severity: high), если `server_tokens off` не задан на уровне http или server

### Fixed

- Ложный `ssl_protocols_weak` на конфигурациях с `TLSv1.2 TLSv1.3` (substring `TLSv1`)
- Ложный `listen_443_no_ssl` / `listen_443_no_http2` для портов вроде `8443` (substring `443`)
- Ложный `ssl_ciphers_weak` при исключениях вида `!MD5` в `ssl_ciphers`
- Ложный `listen_servername_conflict` для `listen 80` и `listen [::]:80` с одним `server_name`
- Ложный location conflict между `/` и дочерними path (`/api`)
- Dead location: учёт server scope и prefix-match без лишнего `/`
- Wildcard-сертификат `*.example.com` покрывает apex `example.com`
- Rate-limit policy не срабатывает, если `limit_req_zone` / `limit_conn_zone` уже определены
- Score/hub summary синхронизированы с policy, cert и module issues через единый список issues

## [1.1.1] — 2026-06-05

### Fixed

- Hub Agents: KPI Critical/Warnings считаются по severity issues, а не по числу offline/warning-агентов
- Hub snapshot: issues breakdown по категориям score (security, performance, …), а не high→security / medium→performance
- Ложное предупреждение «модуль http_proxy не найден» для встроенных модулей nginx (proxy_pass и др.)

## [1.1.0] — 2026-06-05

### Added

- Единый формат вывода CLI (`printSection`, `printTable`, `printIssue`) для analyze, health, score, logs и других команд
- Тесты на ложные срабатывания `listen_servername_conflict` (single file, double include, default vhost)

### Changed

- Сообщение `listen_servername_conflict` указывает оба server-блока и общие listen/server_name

### Fixed

- Ложный `listen_servername_conflict` при повторном include одного vhost и при одиночном server-блоке
- Номера строк server/upstream/location в парсере — строка открытия блока, а не закрывающая `}`
- Шкала Config Score в Hub: метки 0 / 50 / 80 / 100 совпадают с шириной progress bar

## [1.0.0] — 2026-06-05

### Added

- Бинарники `nginx-lens`, `nginx-lens-agent`, `nginx-lens-hub` в одном release-архиве
- Config-first CLI: все параметры в `config.yaml` (секции `validate`, `output`, `route`, `policy` и др.)
- Policy packs: `security-baseline`, `mozilla-ssl`, `performance-baseline`, `caching`, `rate-limit`
- Semantic diff (`nginx-lens diff`), blast-radius, score, ingress-audit
- Hub dashboard с встроенным UI
- CI: coverage gate ≥ 45%, golangci-lint, matrix Linux/macOS/Windows

### Changed

- Проект полностью на Go; PyPI / `pip install` больше не поддерживается
- Обновлена документация: `docs/README.md`, `docs/guide.md`, сокращён справочник конфигурации
- Install scripts и README указывают на `main` и GitHub Releases

### Fixed

- `nginx-lens config validate` принимает все policy packs из `example-config.yaml`
- Синхронизация `isKnownPack` с policy engine
