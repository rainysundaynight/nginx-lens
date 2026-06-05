# Документация nginx-lens

nginx-lens — CLI для анализа, валидации и мониторинга конфигураций Nginx.  
Все параметры задаются в **config.yaml**; отдельные команды принимают аргументы там, где это указано (`route`, `diff`, `explain`).

## Навигация

| Документ | Для кого | Содержание |
|----------|----------|------------|
| [Руководство](guide.md) | Новые пользователи | Установка, init, типовые сценарии, CI |
| [Конфигурация](CONFIGURATION.md) | DevOps / SRE | Секции `config.yaml`, env, Docker, policy |
| [API](API.md) | Интеграции | JSON-схемы CLI, HTTP agent/hub |
| [Web-стек](WEB_DEPLOYMENT.md) | Платформа | Agent, Hub, dashboard, systemd, безопасность |

## Быстрые команды

```bash
nginx-lens init                    # создать config.yaml
nginx-lens config validate         # проверить схему config.yaml
nginx-lens validate                # CI-пайплайн (syntax + analysis + upstream + …)
nginx-lens analyze                 # статический анализ
nginx-lens route --url http://host/path
nginx-lens explain --url http://host/path
```

Полный образец конфига: [example-config.yaml](../example-config.yaml).

## Где лежит config.yaml

1. `NGINX_LENS_CONFIG` — явный путь  
2. `/opt/nginx-lens/config.yaml` — после `sudo nginx-lens init`  
3. `.nginx-lens.yaml` в текущей директории  
4. `~/.nginx-lens/config.yaml` — после `nginx-lens init --user`

## Ограничения парсера

| Режим `parser.mode` | Когда | Ограничения |
|---------------------|-------|-------------|
| `auto` | По умолчанию | `expanded`, если nginx в PATH, иначе `regex` |
| `expanded` | Точный разбор | Нужен рабочий `nginx -t` |
| `regex` | CI без nginx | Не понимает `map`, сложный `if`, dynamic upstream API |

Подробнее — в [руководстве](guide.md#ограничения).
