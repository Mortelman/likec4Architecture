import os
import logging
from datetime import datetime
from typing import Optional

import httpx
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# URL сервиса пользователей — берём из переменной окружения
# В Docker: "http://users-service:8080"
# Локально: "http://localhost:8080"
USERS_SERVICE_URL = os.getenv("USERS_SERVICE_URL", "http://users-service:8080")

# --- Модели ---

class CreateOrderRequest(BaseModel):
    user_id: int
    product: str
    amount: float


class Order(BaseModel):
    id: int
    user_id: int
    product: str
    amount: float
    status: str
    created_at: datetime


# --- Приложение ---

app = FastAPI(
    title="Orders Service",
    description="Сервис заказов. Валидирует user_id через Users Service.",
    version="1.0.0",
)

# In-memory хранилище
orders_db: dict[int, Order] = {}
counter = 0


# --- Вспомогательная функция: вызов Users Service ---

def check_user_exists(user_id: int) -> bool:
    """
    Проверяет существование пользователя через REST вызов к users-service.

    Это межсервисное взаимодействие — ключевой паттерн микросервисов.
    """
    url = f"{USERS_SERVICE_URL}/users/{user_id}"
    logger.info(f"Calling users-service: GET {url}")

    try:
        # Используем httpx для синхронного HTTP запроса
        response = httpx.get(url, timeout=5.0)
        logger.info(f"users-service responded: {response.status_code}")
        return response.status_code == 200
    except httpx.ConnectError:
        logger.error(f"Cannot connect to users-service at {USERS_SERVICE_URL}")
        raise HTTPException(
            status_code=503,
            detail={"code": "USERS_SERVICE_UNAVAILABLE", "message": "Users service is not available"},
        )
    except httpx.TimeoutException:
        logger.error("Request to users-service timed out")
        raise HTTPException(
            status_code=504,
            detail={"code": "USERS_SERVICE_TIMEOUT", "message": "Users service request timed out"},
        )


# --- Эндпоинты ---

@app.get("/health")
def health_check():
    return {"status": "ok"}


@app.get("/orders", response_model=list[Order], tags=["orders"])
def list_orders(user_id: Optional[int] = None):
    """Получить список заказов, опционально фильтруя по user_id"""
    orders = list(orders_db.values())
    if user_id is not None:
        orders = [o for o in orders if o.user_id == user_id]
    return orders


@app.post("/orders", response_model=Order, status_code=201, tags=["orders"])
def create_order(request: CreateOrderRequest):
    """
    Создать заказ.

    Перед созданием проверяет существование пользователя в users-service.
    """
    global counter

    # Валидация через межсервисный вызов
    if not check_user_exists(request.user_id):
        raise HTTPException(
            status_code=404,
            detail={
                "code": "USER_NOT_FOUND",
                "message": f"User with id {request.user_id} not found",
            },
        )

    counter += 1
    order = Order(
        id=counter,
        user_id=request.user_id,
        product=request.product,
        amount=request.amount,
        status="pending",
        created_at=datetime.now(),
    )
    orders_db[order.id] = order
    logger.info(f"Created order {order.id} for user {order.user_id}")
    return order


@app.get("/orders/{order_id}", response_model=Order, tags=["orders"])
def get_order(order_id: int):
    """Получить заказ по ID"""
    if order_id not in orders_db:
        raise HTTPException(
            status_code=404,
            detail={"code": "NOT_FOUND", "message": f"Order {order_id} not found"},
        )
    return orders_db[order_id]
