# 05 — PostgreSQL и Миграции

## PostgreSQL в Docker Compose

Запустить PostgreSQL одной строкой:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: mydb
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data  # Данные не теряются при перезапуске

volumes:
  postgres_data:
```

## Что такое миграции?

**Миграция** — это версионированное изменение схемы базы данных.

### Проблема без миграций
```
v1: таблица users (id, name)
v2: нужно добавить поле email
```
Как синхронизировать схему БД на всех окружениях (dev, staging, prod)?
Как откатить изменение если что-то пошло не так?

### Решение: миграции
```
migrations/
├── 000001_create_users_table.up.sql    ← применить
├── 000001_create_users_table.down.sql  ← откатить
├── 000002_add_email_to_users.up.sql
└── 000002_add_email_to_users.down.sql
```

Каждая миграция имеет:
- **Порядковый номер** — гарантирует правильный порядок применения
- **up** — применить изменение
- **down** — откатить изменение

### Таблица миграций в БД
Инструменты миграций создают служебную таблицу (обычно `schema_migrations`),
которая хранит список применённых миграций:

```sql
SELECT * FROM schema_migrations;
-- version | dirty
-- 1       | false
-- 2       | false
```

## Инструменты миграций

### golang-migrate
Популярный инструмент для Go-проектов (но работает с любыми языками):
```bash
# Применить все миграции
migrate -path ./migrations -database "postgres://..." up

# Откатить последнюю миграцию
migrate -path ./migrations -database "postgres://..." down 1

# Применить до версии 3
migrate -path ./migrations -database "postgres://..." goto 3
```

### Alembic (Python)
Стандарт для Python/SQLAlchemy проектов (рассмотрим в примере 07).

## Типы данных PostgreSQL

```sql
-- Числа
id SERIAL PRIMARY KEY      -- автоинкремент integer
id BIGSERIAL PRIMARY KEY   -- автоинкремент bigint
price NUMERIC(10,2)        -- точные числа (финансы)

-- Строки
name VARCHAR(100)           -- строка до 100 символов
description TEXT            -- строка неограниченной длины

-- Дата и время
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()

-- Булевый
is_active BOOLEAN DEFAULT true

-- UUID
id UUID DEFAULT gen_random_uuid()
```

## SQL для создания таблиц

```sql
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100)  NOT NULL,
    email      VARCHAR(255)  NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индекс для быстрого поиска по email
CREATE INDEX idx_users_email ON users(email);
```

```sql
CREATE TABLE orders (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product    VARCHAR(255)  NOT NULL,
    amount     NUMERIC(10,2) NOT NULL CHECK (amount > 0),
    status     VARCHAR(50)   NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Запуск примера

```bash
# Запустить PostgreSQL и pgAdmin
docker compose up -d

# Подождать запуска (5-10 секунд)
# Запустить миграции вручную
docker compose run --rm migrate

# Подключиться к PostgreSQL через psql
docker compose exec postgres psql -U postgres -d demo_db

# Проверить таблицы
\dt

# Посмотреть структуру таблицы
\d users

# Выйти
\q
```

## pgAdmin

pgAdmin доступен на http://localhost:5050

- Email: `admin@admin.com`
- Password: `admin`

Подключение к серверу:
- Host: `postgres` (имя сервиса в docker-compose!)
- Port: `5432`
- Database: `demo_db`
- Username: `postgres`
- Password: `postgres`

> Обратите внимание: host = `postgres`, а не `localhost`!
> Внутри Docker сети сервисы обращаются друг к другу по **имени сервиса**.

## Полезные psql команды

```sql
\l          -- список баз данных
\c mydb     -- подключиться к базе
\dt         -- список таблиц
\d tablename -- структура таблицы
\di         -- список индексов
```
