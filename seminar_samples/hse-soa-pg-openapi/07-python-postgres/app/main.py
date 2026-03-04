import time
import logging
from datetime import datetime
from fastapi import FastAPI, HTTPException, Depends
from pydantic import BaseModel
from sqlalchemy.orm import Session
from sqlalchemy import text

from app.database import engine, get_db
from app.models import User

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# --- Pydantic схемы (отдельно от SQLAlchemy моделей!) ---

class CreateUserRequest(BaseModel):
    name: str
    email: str


class UserResponse(BaseModel):
    id: int
    name: str
    email: str
    created_at: datetime

    model_config = {"from_attributes": True}  # Позволяет создавать из SQLAlchemy объектов


# --- Приложение ---

app = FastAPI(
    title="Users Service",
    description="Сервис управления пользователями с PostgreSQL",
    version="1.0.0",
)


@app.on_event("startup")
def wait_for_db():
    """Ждём готовности базы данных при запуске"""
    for attempt in range(1, 31):
        try:
            with engine.connect() as conn:
                conn.execute(text("SELECT 1"))
            logger.info("Database is ready")
            return
        except Exception as e:
            logger.info(f"Waiting for database... attempt {attempt}/30: {e}")
            time.sleep(1)
    raise RuntimeError("Cannot connect to database after 30 attempts")


# --- Эндпоинты ---

@app.get("/health")
def health_check():
    return {"status": "ok"}


@app.get("/users", response_model=list[UserResponse], tags=["users"])
def list_users(db: Session = Depends(get_db)):
    """Получить список всех пользователей"""
    return db.query(User).order_by(User.id).all()


@app.post("/users", response_model=UserResponse, status_code=201, tags=["users"])
def create_user(request: CreateUserRequest, db: Session = Depends(get_db)):
    """Создать нового пользователя"""
    # Проверяем уникальность email
    existing = db.query(User).filter(User.email == request.email).first()
    if existing:
        raise HTTPException(
            status_code=409,
            detail={"code": "DUPLICATE_EMAIL", "message": f"Email {request.email} already exists"},
        )

    user = User(name=request.name, email=request.email)
    db.add(user)
    db.commit()
    db.refresh(user)
    return user


@app.get("/users/{user_id}", response_model=UserResponse, tags=["users"])
def get_user(user_id: int, db: Session = Depends(get_db)):
    """Получить пользователя по ID"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=404,
            detail={"code": "NOT_FOUND", "message": f"User with id {user_id} not found"},
        )
    return user


@app.delete("/users/{user_id}", status_code=204, tags=["users"])
def delete_user(user_id: int, db: Session = Depends(get_db)):
    """Удалить пользователя"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=404,
            detail={"code": "NOT_FOUND", "message": f"User with id {user_id} not found"},
        )
    db.delete(user)
    db.commit()
