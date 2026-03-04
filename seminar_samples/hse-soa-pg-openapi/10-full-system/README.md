  # 10 — Полная Система: Go + Python + PostgreSQL

## Финальная архитектура

```
                        Docker Network
    ┌───────────────────────────────────────────────────────┐
    │                                                       │
    │  ┌──────────────────┐      ┌──────────────────┐       │
    │  │  users-service   │      │  orders-service  │       │
    │  │  (Go, :8080)     │◄─────│  (Python, :8000) │       │
    │  │                  │  REST│                  │       │
    │  └────────┬─────────┘      └────────┬─────────┘       │
    │           │ SQL                     │ SQL             │
    │           ▼                         ▼                 │
    │  ┌──────────────────┐      ┌──────────────────┐       │
    │  │  users-db        │      │  orders-db       │       │
    │  │  (PostgreSQL)    │      │  (PostgreSQL)    │       │
    │  └──────────────────┘      └──────────────────┘       │
    │                                                       │
    │  ┌──────────────────┐                                 │
    │  │  swagger-ui      │                                 │
    │  │  (:8090)         │                                 │
    │  └──────────────────┘                                 │
    │                                                       │
    └───────────────────────────────────────────────────────┘
```

**Каждый сервис — своя база данных!**
Это важный принцип микросервисной архитектуры: каждый сервис владеет своими данными.

## Что нового по сравнению с примером 09

1. **PostgreSQL для users-service** — данные не теряются при перезапуске
2. **PostgreSQL для orders-service** — Alembic миграции
3. **healthcheck** для PostgreSQL — сервисы стартуют только когда БД готова
4. **depends_on с condition** — правильное управление порядком запуска
5. **Переменные окружения** — конфигурация через `.env` файл

## Порядок запуска с depends_on

```yaml
services:
  users-service:
    depends_on:
      users-db:
        condition: service_healthy  # Ждём ГОТОВНОСТИ, не просто запуска

  users-db:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5
```

Порядок запуска:
```
1. users-db запускается
2. Healthcheck проверяет готовность pg_isready
3. Только после service_healthy запускается users-service
4. users-service запускает миграции
5. users-service начинает слушать порт 8080
```

## Изолированные сети

В продакшне часто изолируют сети для дополнительной безопасности:

```yaml
networks:
  users-net:     # Только users-service и users-db
  orders-net:    # Только orders-service и orders-db
  api-net:       # Для внешнего доступа (оба сервиса)
```

## Запуск

```bash
# Скопировать переменные окружения
cp .env.example .env

# Запустить всю систему
docker compose up --build

# Проверить что всё запустилось
docker compose ps
```

### Полный сценарий тестирования

```bash
# 1. Создать пользователей
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Иван Иванов", "email": "ivan@example.com"}'

curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Мария Смирнова", "email": "maria@example.com"}'

# 2. Проверить что пользователи сохранились в БД
docker compose exec users-db psql -U postgres -d users_db -c "SELECT * FROM users;"

# 3. Создать заказы
curl -X POST http://localhost:8000/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "product": "Ноутбук", "amount": 89999}'

curl -X POST http://localhost:8000/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 2, "product": "Мышь", "amount": 2999}'

# 4. Проверить заказы в БД
docker compose exec orders-db psql -U postgres -d orders_db -c "SELECT * FROM orders;"

# 5. Обновить статус заказа
curl -X PATCH http://localhost:8000/orders/1 \
  -H "Content-Type: application/json" \
  -d '{"status": "confirmed"}'

# 6. Перезапустить сервисы — данные должны остаться!
docker compose restart
sleep 5
curl http://localhost:8080/users
curl http://localhost:8000/orders

# 7. Посмотреть Swagger UI
# http://localhost:8090
```

### Просмотр логов

```bash
# Все логи
docker compose logs -f

# Только orders-service (видно межсервисные вызовы)
docker compose logs -f orders-service

# Только errors
docker compose logs --since 5m 2>&1 | grep -i error
```

## Что дальше?

После этого семинара рекомендуется изучить:
- **API Gateway** (nginx, Kong, Traefik) — единая точка входа
- **Service Discovery** (Consul, Kubernetes) — автоматическое обнаружение сервисов
- **Circuit Breaker** — защита от каскадных отказов
- **Distributed Tracing** (Jaeger, Zipkin) — трассировка запросов между сервисами
- **Message Queue** (RabbitMQ, Kafka) — асинхронное взаимодействие
