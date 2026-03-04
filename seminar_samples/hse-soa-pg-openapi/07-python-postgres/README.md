# 07 — Python FastAPI с PostgreSQL и Alembic

## Что изучаем

Подключаем PostgreSQL к FastAPI сервису.
Используем **SQLAlchemy** (ORM) для работы с базой данных.
Используем **Alembic** для управления миграциями.

## SQLAlchemy

**SQLAlchemy** — самый популярный Python ORM (Object-Relational Mapper).
Позволяет работать с базой данных через Python объекты вместо сырого SQL.

### Определение модели
```python
from sqlalchemy import Column, BigInteger, String, DateTime
from sqlalchemy.orm import DeclarativeBase
from datetime import datetime, timezone

class Base(DeclarativeBase):
    pass

class User(Base):
    __tablename__ = "users"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(100), nullable=False)
    email = Column(String(255), nullable=False, unique=True)
    created_at = Column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))
```

### Сессия и запросы
```python
from sqlalchemy.orm import Session

# Получить все записи
def get_all_users(db: Session):
    return db.query(User).all()

# Получить по ID
def get_user(db: Session, user_id: int):
    return db.query(User).filter(User.id == user_id).first()

# Создать
def create_user(db: Session, name: str, email: str):
    user = User(name=name, email=email)
    db.add(user)
    db.commit()
    db.refresh(user)  # Обновить объект данными из БД (id, created_at)
    return user

# Удалить
def delete_user(db: Session, user_id: int):
    user = db.query(User).filter(User.id == user_id).first()
    if user:
        db.delete(user)
        db.commit()
        return True
    return False
```

### Dependency Injection в FastAPI
```python
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

engine = create_engine(DATABASE_URL)
SessionLocal = sessionmaker(bind=engine)

def get_db():
    db = SessionLocal()
    try:
        yield db        # Передаём сессию в обработчик
    finally:
        db.close()      # Закрываем после завершения запроса

@app.get("/users")
def list_users(db: Session = Depends(get_db)):  # Inject!
    return db.query(User).all()
```

## Alembic — миграции для Python

**Alembic** — инструмент миграций для SQLAlchemy.
Автоматически создаёт миграции, сравнивая модели с текущей схемой БД.

### Структура
```
alembic/
├── env.py           ← Конфигурация (подключение к БД, импорт моделей)
└── versions/        ← Файлы миграций
    └── 001_initial.py
alembic.ini          ← Настройки alembic
```

### Файл миграции
```python
# alembic/versions/001_create_users.py
def upgrade():
    op.create_table('users',
        sa.Column('id', sa.BigInteger(), nullable=False),
        sa.Column('name', sa.String(100), nullable=False),
        sa.Column('email', sa.String(255), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )

def downgrade():
    op.drop_table('users')
```

### Команды Alembic
```bash
# Применить все миграции
alembic upgrade head

# Откатить последнюю миграцию
alembic downgrade -1

# Создать новую миграцию автоматически (сравнивает модели с БД)
alembic revision --autogenerate -m "add phone column"

# Посмотреть текущую версию
alembic current

# История миграций
alembic history
```

## Запуск

```bash
docker compose up --build
```

Сервис:
1. Дождётся PostgreSQL
2. Запустит Alembic миграции
3. Начнёт принимать запросы

Открой http://localhost:8000/docs

### Тестируем API

```bash
# Создать пользователя
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Пётр Сидоров", "email": "petr@example.com"}'

# Список пользователей
curl http://localhost:8000/users

# Проверить в БД
docker compose exec postgres psql -U postgres -d orders_db -c "SELECT * FROM users;"
```
