# Семинар: OpenAPI + Docker Compose + Go + Python + PostgreSQL

## Программа семинара

| Пример | Тема | Что изучаем |
|--------|------|-------------|
| [01-hello-openapi](./01-hello-openapi/) | OpenAPI спецификация | Структура YAML, paths, schemas, $ref, Swagger UI |
| [02-docker-compose-basics](./02-docker-compose-basics/) | Docker Compose | services, ports, volumes, networks, команды |
| [03-go-http-service](./03-go-http-service/) | Go HTTP сервис | net/http, JSON, routing, Dockerfile multi-stage |
| [04-python-fastapi-service](./04-python-fastapi-service/) | Python FastAPI | Pydantic, автогенерация OpenAPI, Swagger UI |
| [05-postgres-migrations](./05-postgres-migrations/) | PostgreSQL + Миграции | Docker PostgreSQL, golang-migrate, SQL up/down |
| [06-go-postgres](./06-go-postgres/) | Go + PostgreSQL | database/sql, lib/pq, golang-migrate в коде |
| [07-python-postgres](./07-python-postgres/) | Python + PostgreSQL | SQLAlchemy ORM, Alembic миграции |
| [08-openapi-contracts](./08-openapi-contracts/) | OpenAPI контракты | API-First, версионирование, несколько API |
| [09-two-services](./09-two-services/) | Go + Python REST | Межсервисное взаимодействие, Docker DNS |
| [10-full-system](./10-full-system/) | Полная система | Всё вместе: Go + Python + 2x PostgreSQL |

## Быстрый старт

Каждый пример запускается независимо:

```bash
cd 01-hello-openapi
docker compose up
```

## Требования

- Docker Desktop (включает Docker Compose v2)
- curl (для тестирования API)

## Порядок прохождения

Примеры **последовательны** — каждый строится на предыдущем.
Рекомендуется проходить по порядку, читая README в каждой папке.

## Архитектура финального примера

```
Клиент (curl/браузер)
        │
        ├─── GET/POST http://localhost:8080/users ──► users-service (Go)
        │                                                    │
        │                                              users-db (PostgreSQL :5432)
        │
        └─── GET/POST http://localhost:8000/orders ──► orders-service (Python)
                                                              │
                                                    ┌─────────┴──────────┐
                                              orders-db              users-service
                                         (PostgreSQL :5433)      (для валидации user_id)
```
