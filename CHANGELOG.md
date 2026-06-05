# Changelog

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
