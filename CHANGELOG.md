# Changelog

## [1.0.0] — 2026-06-05

### Added

- Бинарники `nginx-lens`, `nginx-lens-agent`, `nginx-lens-hub` в одном release-архиве
- Config-first CLI: все параметры в `config.yaml` (секции `validate`, `output`, `route`, `policy` и др.)
- Policy packs: `security-baseline`, `mozilla-ssl`, `performance-baseline`, `caching`, `rate-limit`
- Semantic diff (`nginx-lens diff`), blast-radius, score, ingress-audit
- Hub dashboard с встроенным UI
- CI: coverage gate ≥ 45%, golangci-lint, matrix Linux/macOS/Windows

### Changed

- Обновлена документация: `docs/README.md`, `docs/guide.md`, сокращён справочник конфигурации
- Install scripts и README указывают на `main` и GitHub Releases

### Fixed

- `nginx-lens config validate` принимает все policy packs из `example-config.yaml`
- Синхронизация `isKnownPack` с policy engine
