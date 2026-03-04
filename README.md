# Marketplace API - Домашнее задание №2

Полнофункциональный REST API для маркетплейса с JWT авторизацией, ролевой моделью доступа и сложной бизнес-логикой заказов.

## 🎯 Реализованные требования

### ✅ Пункты 1-4 (Базовый CRUD)
- [x] OpenAPI спецификация с полным описанием API
- [x] Кодогенерация из OpenAPI (Go server stubs)
- [x] PostgreSQL с миграциями
- [x] CRUD операции для Products
- [x] Мягкое удаление (ARCHIVED status)
- [x] Пагинация и фильтрация
- [x] Индексы на status, category, seller_id

### ✅ Пункты 5-6 (Обработка ошибок и валидация)
- [x] Контрактная обработка ошибок (ErrorResponse)
- [x] Все error codes из спецификации
- [x] Валидация на уровне OpenAPI
- [x] Детальные сообщения об ошибках

### ✅ Пункт 7 (Сложная бизнес-логика заказов)
- [x] Полная реализация создания заказа с 8 шагами:
  1. Ограничение частоты создания (rate limiting)
  2. Проверка активных заказов
  3. Проверка каталога (товары ACTIVE)
  4. Проверка остатков (stock)
  5. Резервирование остатков
  6. Снапшот цен (price_at_order)
  7. Расчёт стоимости с промокодами
  8. Фиксация операции
- [x] Транзакционность (всё или ничего)
- [x] Промокоды (PERCENTAGE/FIXED_AMOUNT)
- [x] Модель состояний заказа

### ✅ Пункт 8 (Логирование API)
- [x] JSON формат логов
- [x] request_id в каждом запросе
- [x] X-Request-Id в заголовках ответа
- [x] Логирование method, endpoint, status_code, duration_ms, user_id, timestamp

### ✅ Пункт 9 (JWT авторизация)
- [x] Регистрация пользователей
- [x] Логин с выдачей access + refresh токенов
- [x] Access token (30 минут)
- [x] Refresh token (7 дней)
- [x] Обновление токенов через /auth/refresh
- [x] Защита всех CRUD эндпоинтов

### ✅ Пункт 10 (Ролевая модель)
- [x] Три роли: USER, SELLER, ADMIN
- [x] Матрица доступа согласно требованиям
- [x] seller_id в products для контроля владения
- [x] Роль в JWT токене (claim "role")
- [x] 403 ACCESS_DENIED при недостаточных правах

## 🏗️ Архитектура

```
marketplace/
├── api/
│   └── openapi.yaml          # OpenAPI 3.1 спецификация
├── migrations/               # SQL миграции
│   ├── 000001_create_products_table.up.sql
│   ├── 000002_create_users_table.up.sql
│   └── 000003_create_orders_tables.up.sql
├── pkg/gen/go/              # Сгенерированный код (gitignore)
├── final_arch_solution/     # LikeC4 диаграммы архитектуры
├── main.go                  # Основное приложение
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 🚀 Быстрый старт

### Предварительные требования
- Docker и Docker Compose
- Go 1.21+ (для локальной разработки)

### 1. Кодогенерация (опционально, уже выполнена)
```bash
./generate.sh
```

### 2. Запуск системы
```bash
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`

### 3. Проверка здоровья
```bash
curl http://localhost:8080/health
```

## 📝 Примеры использования

### Регистрация пользователя
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller@example.com",
    "password": "SecurePass123",
    "role": "SELLER"
  }'
```

### Логин
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller@example.com",
    "password": "SecurePass123"
  }'
```

Ответ:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "user": {
    "id": "uuid",
    "email": "seller@example.com",
    "role": "SELLER",
    "created_at": "2024-03-04T10:00:00Z"
  }
}
```

### Создание товара (SELLER/ADMIN)
```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Смартфон XYZ Pro",
    "description": "Флагманский смартфон",
    "price": 59990.00,
    "stock": 42,
    "category": "Электроника",
    "status": "ACTIVE"
  }'
```

### Получение списка товаров (все роли)
```bash
curl -X GET "http://localhost:8080/api/v1/products?page=0&size=20&status=ACTIVE" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Создание заказа (USER/ADMIN)
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "product_id": "product-uuid",
        "quantity": 2
      }
    ],
    "promo_code": "WINTER2024"
  }'
```

### Создание промокода (SELLER/ADMIN)
```bash
curl -X POST http://localhost:8080/api/v1/promo-codes \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "SUMMER2024",
    "discount_type": "PERCENTAGE",
    "discount_value": 15.00,
    "min_order_amount": 5000.00,
    "max_uses": 100,
    "valid_from": "2024-06-01T00:00:00Z",
    "valid_until": "2024-08-31T23:59:59Z"
  }'
```

## 🧪 Тестирование

### E2E сценарий для защиты

1. **Регистрация трёх пользователей** (USER, SELLER, ADMIN)
2. **Логин SELLER** → создание товаров
3. **Логин USER** → создание заказа
4. **Проверка rate limiting** → повторное создание заказа сразу (должна быть ошибка)
5. **Проверка stock** → заказ с quantity > stock (должна быть ошибка INSUFFICIENT_STOCK)
6. **Проверка промокода** → создание и применение промокода
7. **Проверка ролей** → USER пытается создать товар (403 ACCESS_DENIED)
8. **Проверка владения** → SELLER пытается изменить чужой товар (403)
9. **Мягкое удаление** → DELETE товара → проверка status=ARCHIVED
10. **Просмотр БД** → SELECT из таблиц для подтверждения данных

### Проверка логов
Все запросы логируются в JSON формате:
```json
{
  "request_id": "uuid",
  "method": "POST",
  "endpoint": "/api/v1/orders",
  "status_code": 201,
  "duration_ms": 45,
  "user_id": "user-uuid",
  "timestamp": "2024-03-04T15:30:00Z"
}
```

### Проверка БД
```bash
docker exec -it marketplace-db psql -U postgres -d marketplace_db

# Просмотр товаров
SELECT * FROM products;

# Просмотр заказов
SELECT * FROM orders;

# Просмотр позиций заказа
SELECT * FROM order_items;

# Просмотр промокодов
SELECT * FROM promo_codes;

# Просмотр операций пользователей (rate limiting)
SELECT * FROM user_operations;
```

## 🔒 Безопасность

- Пароли хешируются с помощью bcrypt
- JWT токены подписываются секретным ключом
- Access token короткоживущий (30 минут)
- Refresh token для обновления (7 дней)
- Ролевая модель доступа на уровне API
- Валидация всех входных данных

## 📊 Бизнес-логика заказов

### Создание заказа
1. **Rate limiting**: не более 1 заказа в 5 минут
2. **Активные заказы**: только 1 активный заказ (CREATED/PAYMENT_PENDING)
3. **Валидация товаров**: все товары должны быть ACTIVE
4. **Проверка остатков**: stock >= quantity для каждого товара
5. **Резервирование**: stock -= quantity (транзакционно)
6. **Снапшот цен**: price_at_order фиксируется
7. **Промокоды**: 
   - PERCENTAGE: до 70% скидки
   - FIXED_AMOUNT: фиксированная сумма
   - Проверка min_order_amount, max_uses, valid_from/until
8. **Фиксация**: запись в user_operations

### Модель состояний
```
CREATED → PAYMENT_PENDING → PAID → SHIPPED → COMPLETED
                    ↘
                     CANCELED
```

## 🛠️ Технологии

- **Go 1.21** - основной язык
- **PostgreSQL 16** - база данных
- **golang-migrate** - миграции БД
- **golang-jwt** - JWT токены
- **bcrypt** - хеширование паролей
- **OpenAPI 3.1** - спецификация API
- **Docker & Docker Compose** - контейнеризация

## 📖 API Documentation

Полная документация API доступна в файле `api/openapi.yaml`.

Можно просмотреть через Swagger UI:
```bash
docker run -p 8081:8080 -e SWAGGER_JSON=/api/openapi.yaml -v $(pwd)/api:/api swaggerapi/swagger-ui
```

Затем открыть http://localhost:8081

## 🎓 Автор

Степан Фокин - Домашнее задание №2 по курсу "Сервисно-ориентированные архитектуры"

## 📄 Лицензия

Учебный проект для ВШЭ