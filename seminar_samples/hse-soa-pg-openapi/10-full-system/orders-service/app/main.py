import os
import time
import logging
from datetime import datetime, timezone
from typing import Optional

import httpx
from fastapi import FastAPI, HTTPException, Depends
from pydantic import BaseModel
from sqlalchemy import create_engine, Column, BigInteger, String, Numeric, DateTime, text
from sqlalchemy.orm import DeclarativeBase, sessionmaker, Session

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/orders_db")
USERS_SERVICE_URL = os.getenv("USERS_SERVICE_URL", "http://users-service:8080")

# --- База данных ---

engine = create_engine(DATABASE_URL)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)


class Base(DeclarativeBase):
    pass


class Order(Base):
    __tablename__ = "orders"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, nullable=False)
    product = Column(String(255), nullable=False)
    amount = Column(Numeric(10, 2), nullable=False)
    status = Column(String(50), nullable=False, default="pending")
    created_at = Column(DateTime(timezone=True),
                        default=lambda: datetime.now(timezone.utc), nullable=False)


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


# --- Pydantic схемы ---

class CreateOrderRequest(BaseModel):
    user_id: int
    product: str
    amount: float


class UpdateOrderStatusRequest(BaseModel):
    status: str


class OrderResponse(BaseModel):
    id: int
    user_id: int
    product: str
    amount: float
    status: str
    created_at: datetime

    model_config = {"from_attributes": True}


# --- Приложение ---

app = FastAPI(
    title="Orders Service",
    description="Сервис заказов с PostgreSQL. Валидирует пользователей через Users Service.",
    version="1.0.0",
)


@app.on_event("startup")
def startup():
    # Ждём базу данных
    for attempt in range(1, 31):
        try:
            with engine.connect() as conn:
                conn.execute(text("SELECT 1"))
            logger.info("Database is ready")
            break
        except Exception as e:
            logger.info(f"Waiting for database... {attempt}/30: {e}")
            time.sleep(1)
    else:
        raise RuntimeError("Cannot connect to database")

    # Запускаем миграции через alembic
    import subprocess
    result = subprocess.run(["alembic", "upgrade", "head"], capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"Migrations failed: {result.stderr}")
    logger.info("Migrations applied")


# --- Межсервисное взаимодействие ---

def check_user_exists(user_id: int) -> bool:
    url = f"{USERS_SERVICE_URL}/users/{user_id}"
    logger.info(f"Calling users-service: GET {url}")
    try:
        response = httpx.get(url, timeout=5.0)
        return response.status_code == 200
    except httpx.ConnectError:
        raise HTTPException(503, detail={"code": "USERS_SERVICE_UNAVAILABLE",
                                          "message": "Users service is not available"})
    except httpx.TimeoutException:
        raise HTTPException(504, detail={"code": "USERS_SERVICE_TIMEOUT",
                                          "message": "Users service timed out"})


# --- Эндпоинты ---

@app.get("/health")
def health_check():
    return {"status": "ok"}


@app.get("/orders", response_model=list[OrderResponse], tags=["orders"])
def list_orders(user_id: Optional[int] = None, db: Session = Depends(get_db)):
    query = db.query(Order).order_by(Order.id)
    if user_id is not None:
        query = query.filter(Order.user_id == user_id)
    return query.all()


@app.post("/orders", response_model=OrderResponse, status_code=201, tags=["orders"])
def create_order(request: CreateOrderRequest, db: Session = Depends(get_db)):
    """Создать заказ. Проверяет существование пользователя в users-service."""
    if not check_user_exists(request.user_id):
        raise HTTPException(
            status_code=404,
            detail={"code": "USER_NOT_FOUND",
                    "message": f"User with id {request.user_id} not found"},
        )

    order = Order(
        user_id=request.user_id,
        product=request.product,
        amount=request.amount,
        status="pending",
    )
    db.add(order)
    db.commit()
    db.refresh(order)
    logger.info(f"Created order {order.id} for user {order.user_id}")
    return order


@app.get("/orders/{order_id}", response_model=OrderResponse, tags=["orders"])
def get_order(order_id: int, db: Session = Depends(get_db)):
    order = db.query(Order).filter(Order.id == order_id).first()
    if not order:
        raise HTTPException(404, detail={"code": "NOT_FOUND",
                                          "message": f"Order {order_id} not found"})
    return order


@app.patch("/orders/{order_id}", response_model=OrderResponse, tags=["orders"])
def update_order_status(order_id: int, request: UpdateOrderStatusRequest,
                        db: Session = Depends(get_db)):
    """Обновить статус заказа"""
    valid_statuses = {"pending", "confirmed", "shipped", "delivered", "cancelled"}
    if request.status not in valid_statuses:
        raise HTTPException(400, detail={"code": "INVALID_STATUS",
                                          "message": f"Status must be one of: {valid_statuses}"})

    order = db.query(Order).filter(Order.id == order_id).first()
    if not order:
        raise HTTPException(404, detail={"code": "NOT_FOUND",
                                          "message": f"Order {order_id} not found"})
    order.status = request.status
    db.commit()
    db.refresh(order)
    return order
