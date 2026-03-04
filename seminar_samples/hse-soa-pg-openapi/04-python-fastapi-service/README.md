# 04 — Python FastAPI Сервис

## Что такое FastAPI?

**FastAPI** — современный Python фреймворк для создания REST API.
Главное преимущество: **автоматически генерирует OpenAPI спецификацию** из кода.

Вместо того чтобы писать OpenAPI YAML вручную — пишем Python код с аннотациями типов,
и FastAPI создаёт документацию автоматически.

## Сравнение подходов

| | Ручной OpenAPI | FastAPI |
|---|---|---|
| Документация | Пишем YAML вручную | Генерируется автоматически |
| Синхронизация кода и документации | Нужно следить вручную | Всегда в sync |
| Валидация | Нужно реализовывать | Встроена через Pydantic |
| Swagger UI | Нужно настраивать | Встроен |

## Ключевые компоненты FastAPI

### Pydantic модели (схемы данных)
```python
from pydantic import BaseModel

class CreateUserRequest(BaseModel):
    name: str           # Обязательное поле
    email: str
    age: int | None = None  # Опциональное с default=None
```

Pydantic автоматически:
- Валидирует входящие данные
- Генерирует JSON Schema для OpenAPI
- Преобразует типы (строка "42" → число 42)

### Декораторы маршрутов
```python
app = FastAPI()

@app.get("/users")           # GET /users
def list_users(): ...

@app.post("/users")          # POST /users
def create_user(): ...

@app.get("/users/{user_id}") # GET /users/123
def get_user(user_id: int): ...
```

### Path и Query параметры
```python
@app.get("/users/{user_id}")
def get_user(
    user_id: int,           # Path parameter — из URL
    include_orders: bool = False,  # Query parameter — ?include_orders=true
):
    ...
```

### Request body
```python
@app.post("/users")
def create_user(request: CreateUserRequest):  # Pydantic модель = тело запроса
    # request.name, request.email — уже провалидированы!
    ...
```

### HTTP статус коды и ошибки
```python
from fastapi import HTTPException

@app.get("/users/{user_id}")
def get_user(user_id: int):
    if user_id not in store:
        raise HTTPException(
            status_code=404,
            detail={"code": "NOT_FOUND", "message": f"User {user_id} not found"}
        )
```

## Автодокументация

FastAPI предоставляет два UI:
- **Swagger UI**: http://localhost:8000/docs
- **ReDoc**: http://localhost:8000/redoc
- **OpenAPI JSON**: http://localhost:8000/openapi.json

## Запуск

```bash
docker compose up --build
```

Открой http://localhost:8000/docs — интерактивная документация API!

### Тестируем API

```bash
# Создать пользователя
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Мария Смирнова", "email": "maria@example.com"}'

# Список пользователей
curl http://localhost:8000/users

# Получить пользователя
curl http://localhost:8000/users/1

# Health check
curl http://localhost:8000/health
```

## Dockerfile для Python

```dockerfile
FROM python:3.12-slim

WORKDIR /app

# Сначала зависимости (кэшируется Docker'ом)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

> **uvicorn** — ASGI сервер для запуска FastAPI приложений
