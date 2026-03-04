# 02 — Основы Docker Compose

## Что такое Docker Compose?

**Docker Compose** — инструмент для запуска многоконтейнерных приложений.
Вместо того чтобы запускать каждый контейнер отдельной командой, описываем всё в одном файле `docker-compose.yml`.

## Зачем нужен Docker Compose?

Типичное приложение состоит из:
- Веб-сервер (Go/Python/Node.js)
- База данных (PostgreSQL, MySQL)
- Кэш (Redis)
- Очередь сообщений (RabbitMQ, Kafka)

Без Compose нужно вручную:
1. Запустить каждый контейнер
2. Создать сеть между ними
3. Передать переменные окружения
4. Смонтировать volumes

С Compose: один файл + одна команда.

## Структура docker-compose.yml

```yaml
services:           # Список сервисов (контейнеров)
  service-name:
    image: ...      # Готовый образ из Docker Hub
    build: ...      # ИЛИ собрать из Dockerfile
    ports:          # Проброс портов host:container
    environment:    # Переменные окружения
    volumes:        # Монтирование файлов/папок
    depends_on:     # Зависимости от других сервисов
    networks:       # К каким сетям подключён

networks:           # Определение сетей (опционально)
  my-network:

volumes:            # Именованные тома (опционально)
  my-volume:
```

## Ключевые понятия

### Сети (Networks)
По умолчанию все сервисы в одном `docker-compose.yml` находятся в одной сети.
Сервисы обращаются друг к другу **по имени сервиса**:

```yaml
services:
  web:
    image: nginx
    # Этот сервис доступен как "web" внутри docker сети
  app:
    image: myapp
    environment:
      # Обращаемся к nginx по имени сервиса, не по IP!
      NGINX_URL: http://web:80
```

### Порты (Ports)
```yaml
ports:
  - "8080:80"   # host_port:container_port
  # Запросы на localhost:8080 попадут на порт 80 контейнера
```

### Volumes
```yaml
volumes:
  - ./local/path:/container/path    # Bind mount (папка с хоста)
  - my-volume:/data                 # Named volume (управляется Docker)
  - ./config.yml:/app/config.yml:ro # :ro = read-only
```

### Environment
```yaml
environment:
  - DATABASE_URL=postgres://user:pass@db:5432/mydb
  - DEBUG=true
# ИЛИ через файл:
env_file:
  - .env
```

### depends_on
```yaml
services:
  app:
    depends_on:
      - db      # Запустить db ПЕРЕД app
      # Важно: depends_on ждёт только запуска контейнера,
      # но НЕ готовности приложения внутри!
```

## Основные команды

```bash
# Запустить все сервисы (в фоне)
docker compose up -d

# Запустить с пересборкой образов
docker compose up -d --build

# Остановить все сервисы
docker compose down

# Остановить И удалить volumes
docker compose down -v

# Посмотреть логи
docker compose logs
docker compose logs -f app    # Следить за логами сервиса app

# Статус контейнеров
docker compose ps

# Выполнить команду внутри контейнера
docker compose exec app bash
docker compose exec db psql -U postgres
```

## Запуск примера

```bash
docker compose up -d
```

В этом примере запускаются:
- **nginx** на порту 8080 — простой веб-сервер
- **whoami** на порту 8081 — сервис, показывающий информацию о запросе

Открой:
- http://localhost:8080 — страница nginx
- http://localhost:8081 — информация о контейнере whoami

```bash
# Посмотреть статус
docker compose ps

# Посмотреть логи nginx
docker compose logs nginx

# Остановить
docker compose down
```