from datetime import datetime
from typing import Optional
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, EmailStr

# --- Модели данных ---

class CreateUserRequest(BaseModel):
    name: str
    email: str


class User(BaseModel):
    id: int
    name: str
    email: str
    created_at: datetime


class ErrorResponse(BaseModel):
    code: str
    message: str


# --- Приложение ---

app = FastAPI(
    title="Users Service",
    description="Сервис управления пользователями (in-memory хранилище)",
    version="1.0.0",
)

# In-memory хранилище
users_db: dict[int, User] = {}
counter = 0


# --- Эндпоинты ---

@app.get("/health")
def health_check():
    return {"status": "ok"}


@app.get("/users", response_model=list[User], tags=["users"])
def list_users():
    """Получить список всех пользователей"""
    return list(users_db.values())


@app.post("/users", response_model=User, status_code=201, tags=["users"])
def create_user(request: CreateUserRequest):
    """
    Создать нового пользователя.

    - **name**: имя пользователя (обязательно)
    - **email**: email адрес (обязательно)
    """
    global counter
    counter += 1

    user = User(
        id=counter,
        name=request.name,
        email=request.email,
        created_at=datetime.now(),
    )
    users_db[user.id] = user
    return user


@app.get("/users/{user_id}", response_model=User, tags=["users"])
def get_user(user_id: int):
    """Получить пользователя по ID"""
    if user_id not in users_db:
        raise HTTPException(
            status_code=404,
            detail={"code": "NOT_FOUND", "message": f"User with id {user_id} not found"},
        )
    return users_db[user_id]


@app.delete("/users/{user_id}", status_code=204, tags=["users"])
def delete_user(user_id: int):
    """Удалить пользователя"""
    if user_id not in users_db:
        raise HTTPException(
            status_code=404,
            detail={"code": "NOT_FOUND", "message": f"User with id {user_id} not found"},
        )
    del users_db[user_id]
