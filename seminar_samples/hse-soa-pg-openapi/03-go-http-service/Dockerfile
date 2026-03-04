# Этап 1: Сборка бинарника
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Сначала копируем файлы зависимостей — Docker кэширует этот слой
# Повторная сборка при изменении кода НЕ будет скачивать зависимости заново
COPY go.mod ./
RUN go mod download

# Копируем исходный код и собираем
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Этап 2: Минимальный финальный образ
# alpine ~5MB vs golang ~300MB
FROM alpine:3.19

WORKDIR /app

# Копируем только скомпилированный бинарник из этапа builder
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
