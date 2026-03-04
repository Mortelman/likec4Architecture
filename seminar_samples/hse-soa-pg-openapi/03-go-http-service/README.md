# 03 — Go HTTP Сервис

## Что изучаем

Пишем простой HTTP-сервис на Go без базы данных — данные хранятся в памяти.
Цель: понять структуру Go-сервиса, Dockerfile и Docker Compose для него.

## Структура Go HTTP сервера

```
main.go
├── Модели данных (struct)
├── Хранилище (in-memory slice/map)
├── Обработчики (handlers)
└── Роутер + запуск сервера
```

## Как устроен HTTP сервер на Go

```go
// Стандартная библиотека net/http
http.HandleFunc("/path", handlerFunc)
http.ListenAndServe(":8080", nil)

// Обработчик
func handlerFunc(w http.ResponseWriter, r *http.Request) {
    // w — для записи ответа
    // r — входящий запрос (метод, URL, тело, заголовки)
}
```

### Чтение тела запроса
```go
var req CreateUserRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```

### Запись JSON ответа
```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)   // 201
json.NewEncoder(w).Encode(user)
```

### Извлечение path параметра
```go
// URL: /users/42
// r.PathValue("id") — доступно в Go 1.22+
idStr := r.PathValue("id")
id, err := strconv.ParseInt(idStr, 10, 64)
```

## Dockerfile для Go

```dockerfile
# Многоэтапная сборка (multi-stage build)

# Этап 1: Сборка
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download          # Скачать зависимости (кэшируется!)
COPY . .
RUN go build -o server .    # Собрать бинарник

# Этап 2: Финальный образ
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .   # Только бинарник
EXPOSE 8080
CMD ["./server"]
```

**Почему multi-stage?**
- Образ builder: ~300MB (Go компилятор + зависимости)
- Финальный образ: ~10MB (только бинарник + alpine)

## Запуск

```bash
docker compose up --build
```

### Тестируем API

```bash
# Создать пользователя
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Иван Иванов", "email": "ivan@example.com"}'

# Получить всех пользователей
curl http://localhost:8080/users

# Получить пользователя по ID
curl http://localhost:8080/users/1

# Удалить пользователя
curl -X DELETE http://localhost:8080/users/1

# Health check
curl http://localhost:8080/health
```

## Важные моменты

> **Данные хранятся в памяти!** При перезапуске контейнера все данные потеряются.
> В следующих примерах подключим PostgreSQL.

> В Go 1.22+ появился улучшенный роутер в `net/http` с поддержкой path параметров.
> В более ранних версиях использовали библиотеки: gorilla/mux, chi, gin.
