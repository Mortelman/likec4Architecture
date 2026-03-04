# 06 — Go Сервис с PostgreSQL

## Что изучаем

Подключаем PostgreSQL к Go сервису.
Добавляем запуск миграций при старте приложения.

## Подключение к PostgreSQL из Go

Используем стандартный пакет `database/sql` с драйвером `lib/pq`.

```go
import (
    "database/sql"
    _ "github.com/lib/pq"  // Регистрирует драйвер "postgres"
)

db, err := sql.Open("postgres", "postgres://user:pass@host:5432/dbname?sslmode=disable")
```

### Connection String
```
postgres://[user]:[password]@[host]:[port]/[database]?[options]

postgres://postgres:postgres@localhost:5432/users_db?sslmode=disable
```

### Ожидание готовности базы данных
```go
// PostgreSQL запускается не мгновенно!
// Нужно подождать, пока он будет готов принимать подключения
for i := 0; i < 30; i++ {
    if err = db.Ping(); err == nil {
        break
    }
    log.Printf("Waiting for database... attempt %d/30", i+1)
    time.Sleep(time.Second)
}
```

## CRUD операции с database/sql

### Чтение нескольких строк (SELECT)
```go
rows, err := db.QueryContext(ctx, "SELECT id, name, email, created_at FROM users")
defer rows.Close()

var users []User
for rows.Next() {
    var u User
    err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
    // ...
    users = append(users, u)
}
```

### Чтение одной строки
```go
var u User
err := db.QueryRowContext(ctx,
    "SELECT id, name, email, created_at FROM users WHERE id = $1",
    id,
).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)

if err == sql.ErrNoRows {
    // Запись не найдена
}
```

### Вставка (INSERT)
```go
var newID int64
err := db.QueryRowContext(ctx,
    "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
    name, email,
).Scan(&newID)
```

### Удаление (DELETE)
```go
result, err := db.ExecContext(ctx,
    "DELETE FROM users WHERE id = $1", id)

rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
    // Запись не найдена
}
```

> **$1, $2** — плейсхолдеры в PostgreSQL (в MySQL используется ?)
> ВСЕГДА используй плейсхолдеры вместо конкатенации строк — защита от SQL injection!

## Запуск миграций из Go кода

```go
import "github.com/golang-migrate/migrate/v4"

m, err := migrate.New(
    "file://migrations",
    "postgres://...",
)
m.Up() // Применить все новые миграции
```

Это удобнее чем отдельный контейнер — миграции запускаются автоматически при старте сервиса.

## Запуск

```bash
docker compose up --build
```

Сервис сам:
1. Дождётся запуска PostgreSQL
2. Запустит миграции
3. Начнёт принимать запросы

### Тестируем API

```bash
# Создать пользователя
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Алексей Козлов", "email": "alexey@example.com"}'

# Список пользователей
curl http://localhost:8080/users

# Получить пользователя
curl http://localhost:8080/users/1

# Удалить пользователя
curl -X DELETE http://localhost:8080/users/1

# Проверить в базе напрямую
docker compose exec postgres psql -U postgres -d users_db -c "SELECT * FROM users;"
```

### Остановить и проверить что данные сохранились

```bash
docker compose down      # Останавливаем (НЕ удаляем volumes)
docker compose up -d     # Запускаем снова
curl http://localhost:8080/users  # Данные на месте!

docker compose down -v   # Останавливаем И удаляем volumes (данные потеряются)
```
