# 01 — Знакомство с OpenAPI

## Что такое OpenAPI?

**OpenAPI Specification (OAS)** — это стандарт описания REST API в машинночитаемом формате (YAML или JSON).
Раньше назывался **Swagger**, сейчас правильное название — OpenAPI 3.x.

Зачем нужен:
- Документация API, которую понимают и люди, и машины
- Генерация клиентских SDK на любом языке
- Генерация серверного кода (заглушки)
- Валидация запросов/ответов
- Единый контракт между командами фронтенда и бэкенда

## Структура OpenAPI 3.0 документа

```yaml
openapi: 3.0.3          # Версия спецификации

info:                   # Метаданные API
  title: ...
  version: ...

servers:                # Список серверов
  - url: http://localhost:8080

paths:                  # Эндпоинты (маршруты)
  /resource:
    get: ...
    post: ...

components:             # Переиспользуемые компоненты
  schemas: ...          # Модели данных
  responses: ...
  parameters: ...
```

## Ключевые понятия

### Path Item
Описывает один URL и операции над ним:
```yaml
paths:
  /users:
    get:               # GET /users
      summary: Список пользователей
    post:              # POST /users
      summary: Создать пользователя
  /users/{id}:         # {id} — path parameter
    get: ...
```

### Operation
Одна HTTP-операция (GET, POST, PUT, DELETE, PATCH):
```yaml
get:
  summary: Краткое описание
  description: Подробное описание
  operationId: listUsers    # Уникальный идентификатор операции
  tags:                     # Группировка в UI
    - users
  parameters: [...]         # Query/path/header параметры
  requestBody: ...          # Тело запроса
  responses:                # Возможные ответы
    '200':
      description: OK
```

### Schema (JSON Schema)
Описание структуры данных:
```yaml
components:
  schemas:
    User:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
          example: 1
        name:
          type: string
          minLength: 1
          maxLength: 100
          example: "Иван Иванов"
        email:
          type: string
          format: email
```

### $ref — ссылки на компоненты
```yaml
# Вместо копирования схемы везде используем ссылку:
responses:
  '200':
    content:
      application/json:
        schema:
          $ref: '#/components/schemas/User'
```

### Типы параметров
```yaml
parameters:
  - name: id
    in: path      # В пути: /users/{id}
    required: true
    schema:
      type: integer

  - name: limit
    in: query     # В query string: /users?limit=10
    schema:
      type: integer
      default: 10

  - name: X-API-Key
    in: header    # В заголовке
    schema:
      type: string
```

## Запуск примера

В этой папке — OpenAPI спецификация для простого TODO API и Swagger UI для её просмотра.

```bash
docker compose up
```

Открой в браузере: **http://localhost:8080**

Там ты увидишь интерактивную документацию. Можно:
- Изучать эндпоинты
- Смотреть схемы данных
- Нажимать "Try it out" (но реального сервера нет — только документация)

## Что изучить в openapi.yaml

1. Структуру `info`, `servers`, `paths`
2. Как описаны GET и POST операции
3. Как `requestBody` описывает тело запроса
4. Как `$ref` ссылается на общие схемы в `components`
5. Коды ответов: 200, 201, 400, 404