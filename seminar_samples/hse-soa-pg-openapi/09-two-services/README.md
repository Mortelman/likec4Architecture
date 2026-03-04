# 09 — Два Сервиса: Go + Python через REST

## Архитектура

```
                    Docker Network
    ┌──────────────────────────────────────────┐
    │                                          │
    │  ┌─────────────────┐                     │
    │  │  users-service  │                     │
    │  │  (Go, :8080)    │                     │
    │  │                 │                     │
    │  │  GET  /users    │◄──────────────┐     │
    │  │  POST /users    │               │     │
    │  │  GET  /users/id │               │     │
    │  └─────────────────┘               │     │
    │                                    │     │
    │  ┌─────────────────┐               │     │
    │  │  orders-service │  Проверяет    │     │
    │  │  (Python, :8000)│  user_id ─────┘     │
    │  │                 │                     │
    │  │  GET  /orders   │                     │
    │  │  POST /orders   │                     │
    │  └─────────────────┘                     │
    │                                          │
    └──────────────────────────────────────────┘
           ↑                    ↑
    localhost:8080        localhost:8000
    (с хоста)             (с хоста)
```

## Ключевой момент: DNS в Docker Compose

Внутри Docker сети сервисы находят друг друга по **имени сервиса**:

```yaml
# docker-compose.yml
services:
  users-service:    # ← это имя работает как hostname!
    ...
  orders-service:
    environment:
      # orders-service обращается к users-service по имени:
      USERS_SERVICE_URL: http://users-service:8080
```

> Это работает через встроенный DNS Docker.
> Никаких IP адресов — только имена сервисов!

## HTTP клиент в Python (httpx)

```python
import httpx
import os

USERS_SERVICE_URL = os.getenv("USERS_SERVICE_URL", "http://users-service:8080")

def check_user_exists(user_id: int) -> bool:
    try:
        response = httpx.get(f"{USERS_SERVICE_URL}/users/{user_id}", timeout=5.0)
        return response.status_code == 200
    except httpx.RequestError:
        raise HTTPException(503, detail="Users service unavailable")
```

## HTTP клиент в Go

```go
import "net/http"

usersServiceURL := os.Getenv("USERS_SERVICE_URL")

func checkUserExists(userID int64) (bool, error) {
    resp, err := http.Get(fmt.Sprintf("%s/users/%d", usersServiceURL, userID))
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK, nil
}
```

## Паттерн: проверка существования ресурса

При создании заказа orders-service должен убедиться что пользователь существует.
Это типичный паттерн в микросервисной архитектуре:

```
1. Клиент → POST /orders {user_id: 42}
2. orders-service → GET users-service/users/42
3. Если 404 → вернуть 404 с USER_NOT_FOUND
4. Если 200 → создать заказ
5. orders-service → Клиент 201 Created
```

## Запуск

```bash
docker compose up --build
```

### Тестируем взаимодействие

```bash
# 1. Создаём пользователя в users-service
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Анна Белова", "email": "anna@example.com"}'
# → {"id": 1, ...}

# 2. Создаём заказ для пользователя (orders-service → users-service)
curl -X POST http://localhost:8000/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "product": "Курс SOA", "amount": 5000}'
# → {"id": 1, "user_id": 1, ...}

# 3. Попробуем создать заказ для несуществующего пользователя
curl -X POST http://localhost:8000/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 999, "product": "Тест", "amount": 100}'
# → 404 {"code": "USER_NOT_FOUND", "message": "User 999 not found"}

# 4. Посмотреть логи — видно межсервисный вызов
docker compose logs orders-service
```

## Данные в памяти

> В этом примере данные по-прежнему хранятся в памяти.
> В следующем (финальном) примере добавим PostgreSQL для обоих сервисов.
