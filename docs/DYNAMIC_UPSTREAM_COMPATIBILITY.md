# Совместимость с ngx_dynamic_upstream

## Анализ совместимости

`nginx-lens` **полностью совместим** с модулем [`ngx_dynamic_upstream`](https://github.com/ZigzagAK/ngx_dynamic_upstream) в большинстве случаев.

### ✅ Что работает без изменений

1. **Обычные upstream блоки с серверами в конфиге**
   ```nginx
   upstream backends {
       zone zone_for_backends 1m;
       server 127.0.0.1:6001;
       server 127.0.0.1:6002;
   }
   ```
   - ✅ Парсятся корректно
   - ✅ Команды `health` и `resolve` работают
   - ✅ Новые директивы (`zone`) игнорируются парсером, но не мешают

2. **Upstream блоки с DNS background updates**
   ```nginx
   upstream mail {
       zone mail 1m;
       dns_update 60s;
       dns_ipv6 off;
       server mail.ru;
       server google.com backup;
   }
   ```
   - ✅ Серверы парсятся корректно
   - ✅ DNS резолвинг работает через `nginx-lens resolve`
   - ✅ Новые директивы (`dns_update`, `dns_ipv6`) не влияют на парсинг

3. **Комбинированные upstream блоки**
   - Можно иметь серверы в конфиге + добавлять через API
   - Серверы из конфига будут проверяться через `nginx-lens health`

### ⚠️ Ограничения

**Upstream блоки только с `dynamic_state_file` (без директив `server`)**

```nginx
upstream backends2 {
    zone zone_for_backends2 1m;
    dynamic_state_file backend.peers;
}
```

**Проблема:**
- Серверы добавляются динамически через HTTP API модуля
- В конфиге нет директив `server`
- `nginx-lens` не может найти серверы для проверки

**Текущее поведение:**
- Парсер корректно обрабатывает такой блок (не падает)
- Возвращает пустой список серверов `[]`
- Команды `health` и `resolve` пропускают такой upstream

**Пример вывода:**
```bash
$ nginx-lens health /etc/nginx/nginx.conf
# Upstream 'backends2' не будет показан, так как серверы не найдены
```

## Реализованное решение: Интеграция с API модуля ✅

Начиная с версии 0.6.0, `nginx-lens` поддерживает интеграцию с HTTP API модуля `ngx_dynamic_upstream`.

**Преимущества:**
- ✅ Полная поддержка динамических upstream
- ✅ Актуальные данные о серверах
- ✅ Можно проверять серверы, добавленные через API
- ✅ Автоматическое обогащение upstream серверами из API

**Требования:**
- Настроенный endpoint для API (например, `/dynamic`)
- Доступ к nginx по HTTP
- Правильная настройка allow/deny в location блоке

**Конфигурация:**
```yaml
# config.yaml или /opt/nginx-lens/config.yaml
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

## Использование

### Настройка

1. **Включите интеграцию в конфиге:**
   ```yaml
   dynamic_upstream:
     enabled: true
     api_url: http://127.0.0.1:6000/dynamic
     timeout: 2.0
   ```

2. **Убедитесь, что API endpoint доступен:**
   ```bash
   curl "http://127.0.0.1:6000/dynamic?upstream=backends2"
   ```

3. **Используйте команды как обычно:**
   ```bash
   nginx-lens health /etc/nginx/nginx.conf
   nginx-lens resolve /etc/nginx/nginx.conf
   nginx-lens validate /etc/nginx/nginx.conf
   ```

### Обработка ошибок

Если API недоступен или возвращает ошибку:
- Команды продолжают работу с серверами из конфига
- Динамические upstream без серверов в конфиге пропускаются
- Ошибки логируются, но не прерывают выполнение

## Рекомендации

### Для текущего использования

1. **Если upstream содержит серверы в конфиге** - всё работает как обычно
2. **Если upstream только с `dynamic_state_file`** - используйте API модуля напрямую:
   ```bash
   # Получить список серверов
   curl "http://127.0.0.1:6000/dynamic?upstream=backends2"
   
   # Проверить конкретный сервер
   curl "http://127.0.0.1:6000/dynamic?upstream=backends2&server=127.0.0.1:6001&verbose="
   ```

### Для будущего развития

Рекомендуется реализовать **Вариант 1** (интеграция с API) для полной поддержки динамических upstream. Это можно сделать в отдельной команде или опциональной функции.

## Примеры конфигураций

### Пример 1: Смешанная конфигурация
```nginx
http {
    # Статические серверы - проверяются через nginx-lens
    upstream backends {
        zone zone_for_backends 1m;
        server 127.0.0.1:6001;
        server 127.0.0.1:6002;
    }
    
    # Динамические серверы - не проверяются через nginx-lens
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
        
        location / {
            proxy_pass http://backends;
        }
    }
}
```

**Использование:**
```bash
# Проверка статических серверов
nginx-lens health /etc/nginx/nginx.conf
# Проверит только upstream 'backends'

# Проверка динамических серверов через API
curl "http://127.0.0.1:6000/dynamic?upstream=backends2&verbose="
```

### Пример 2: DNS background updates
```nginx
upstream mail {
    zone mail 1m;
    dns_update 60s;
    dns_ipv6 off;
    server mail.ru;
    server google.com backup;
}
```

**Использование:**
```bash
# DNS резолвинг работает
nginx-lens resolve /etc/nginx/nginx.conf
# Покажет резолвленные IP для mail.ru и google.com

# Health check работает
nginx-lens health /etc/nginx/nginx.conf
# Проверит доступность mail.ru и google.com
```

## Заключение

**Текущая совместимость: ✅ Хорошая**

- Большинство случаев работают без изменений
- Новые директивы модуля не ломают парсинг
- Единственное ограничение - upstream блоки только с `dynamic_state_file`

**Рекомендация:** Для полной поддержки можно добавить интеграцию с HTTP API модуля в будущих версиях.

