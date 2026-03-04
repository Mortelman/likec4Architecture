#!/bin/bash

# E2E тестовый скрипт для демонстрации на защите
# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  Marketplace API - E2E Test Script${NC}"
echo -e "${YELLOW}========================================${NC}\n"

# Функция для красивого вывода
print_step() {
    echo -e "\n${GREEN}>>> $1${NC}"
}

print_error() {
    echo -e "${RED}ERROR: $1${NC}"
}

# Проверка здоровья сервиса
print_step "1. Проверка здоровья сервиса"
curl -s $BASE_URL/health | jq .

# Регистрация пользователей
print_step "2. Регистрация SELLER"
SELLER_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller@test.com",
    "password": "SecurePass123",
    "role": "SELLER"
  }')
echo $SELLER_RESPONSE | jq .

print_step "3. Регистрация USER"
USER_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@test.com",
    "password": "SecurePass123",
    "role": "USER"
  }')
echo $USER_RESPONSE | jq .

print_step "4. Регистрация ADMIN"
ADMIN_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@test.com",
    "password": "SecurePass123",
    "role": "ADMIN"
  }')
echo $ADMIN_RESPONSE | jq .

# Логин пользователей
print_step "5. Логин SELLER"
SELLER_LOGIN=$(curl -s -X POST $BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller@test.com",
    "password": "SecurePass123"
  }')
SELLER_TOKEN=$(echo $SELLER_LOGIN | jq -r '.access_token')
echo "Access Token: ${SELLER_TOKEN:0:50}..."

print_step "6. Логин USER"
USER_LOGIN=$(curl -s -X POST $BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@test.com",
    "password": "SecurePass123"
  }')
USER_TOKEN=$(echo $USER_LOGIN | jq -r '.access_token')
echo "Access Token: ${USER_TOKEN:0:50}..."

print_step "7. Логин ADMIN"
ADMIN_LOGIN=$(curl -s -X POST $BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@test.com",
    "password": "SecurePass123"
  }')
ADMIN_TOKEN=$(echo $ADMIN_LOGIN | jq -r '.access_token')
echo "Access Token: ${ADMIN_TOKEN:0:50}..."

# Создание товаров (SELLER)
print_step "8. SELLER создаёт товар"
PRODUCT1=$(curl -s -X POST $BASE_URL/api/v1/products \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Смартфон XYZ Pro",
    "description": "Флагманский смартфон с 6.7 дисплеем",
    "price": 59990.00,
    "stock": 10,
    "category": "Электроника",
    "status": "ACTIVE"
  }')
PRODUCT1_ID=$(echo $PRODUCT1 | jq -r '.id')
echo $PRODUCT1 | jq .

print_step "9. SELLER создаёт второй товар"
PRODUCT2=$(curl -s -X POST $BASE_URL/api/v1/products \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Наушники ABC",
    "description": "Беспроводные наушники с шумоподавлением",
    "price": 15990.00,
    "stock": 5,
    "category": "Электроника",
    "status": "ACTIVE"
  }')
PRODUCT2_ID=$(echo $PRODUCT2 | jq -r '.id')
echo $PRODUCT2 | jq .

# Проверка ролей - USER пытается создать товар
print_step "10. USER пытается создать товар (должна быть ошибка 403)"
curl -s -X POST $BASE_URL/api/v1/products \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Тест",
    "price": 100,
    "stock": 1,
    "category": "Тест",
    "status": "ACTIVE"
  }' | jq .

# Получение списка товаров
print_step "11. Получение списка товаров (все роли могут)"
curl -s -X GET "$BASE_URL/api/v1/products?page=0&size=20&status=ACTIVE" \
  -H "Authorization: Bearer $USER_TOKEN" | jq .

# Создание промокода (SELLER/ADMIN)
print_step "12. SELLER создаёт промокод"
PROMO=$(curl -s -X POST $BASE_URL/api/v1/promo-codes \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "WINTER2024",
    "discount_type": "PERCENTAGE",
    "discount_value": 10.00,
    "min_order_amount": 10000.00,
    "max_uses": 100,
    "valid_from": "2024-01-01T00:00:00Z",
    "valid_until": "2024-12-31T23:59:59Z"
  }')
echo $PROMO | jq .

# Создание заказа (USER)
print_step "13. USER создаёт заказ с промокодом"
ORDER1=$(curl -s -X POST $BASE_URL/api/v1/orders \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {
        \"product_id\": \"$PRODUCT1_ID\",
        \"quantity\": 2
      }
    ],
    \"promo_code\": \"WINTER2024\"
  }")
ORDER1_ID=$(echo $ORDER1 | jq -r '.id')
echo $ORDER1 | jq .

# Проверка rate limiting
print_step "14. USER пытается создать второй заказ сразу (должна быть ошибка 429)"
curl -s -X POST $BASE_URL/api/v1/orders \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {
        \"product_id\": \"$PRODUCT2_ID\",
        \"quantity\": 1
      }
    ]
  }" | jq .

# Проверка недостаточного stock
print_step "15. Попытка заказать больше, чем есть на складе (должна быть ошибка INSUFFICIENT_STOCK)"
sleep 6  # Ждём, чтобы обойти rate limiting
curl -s -X POST $BASE_URL/api/v1/orders \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"items\": [
      {
        \"product_id\": \"$PRODUCT2_ID\",
        \"quantity\": 100
      }
    ]
  }" | jq .

# Получение заказа
print_step "16. USER получает свой заказ"
curl -s -X GET "$BASE_URL/api/v1/orders/$ORDER1_ID" \
  -H "Authorization: Bearer $USER_TOKEN" | jq .

# Проверка владения - SELLER пытается изменить чужой товар
print_step "17. Создаём второго SELLER"
SELLER2_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller2@test.com",
    "password": "SecurePass123",
    "role": "SELLER"
  }')

SELLER2_LOGIN=$(curl -s -X POST $BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller2@test.com",
    "password": "SecurePass123"
  }')
SELLER2_TOKEN=$(echo $SELLER2_LOGIN | jq -r '.access_token')

print_step "18. SELLER2 пытается изменить товар SELLER1 (должна быть ошибка 403)"
curl -s -X PUT "$BASE_URL/api/v1/products/$PRODUCT1_ID" \
  -H "Authorization: Bearer $SELLER2_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Взломанный товар",
    "price": 1.00,
    "stock": 999,
    "category": "Электроника",
    "status": "ACTIVE"
  }' | jq .

# Мягкое удаление
print_step "19. SELLER удаляет свой товар (мягкое удаление)"
curl -s -X DELETE "$BASE_URL/api/v1/products/$PRODUCT2_ID" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -w "\nHTTP Status: %{http_code}\n"

print_step "20. Проверяем, что товар в статусе ARCHIVED"
curl -s -X GET "$BASE_URL/api/v1/products/$PRODUCT2_ID" \
  -H "Authorization: Bearer $SELLER_TOKEN" | jq .

# Итоговая статистика
print_step "21. Итоговая статистика"
echo -e "\n${YELLOW}=== Созданные ресурсы ===${NC}"
echo "SELLER Token: ${SELLER_TOKEN:0:30}..."
echo "USER Token: ${USER_TOKEN:0:30}..."
echo "Product 1 ID: $PRODUCT1_ID"
echo "Product 2 ID: $PRODUCT2_ID"
echo "Order ID: $ORDER1_ID"

echo -e "\n${YELLOW}=== Для проверки БД ===${NC}"
echo "docker exec -it marketplace-db psql -U postgres -d marketplace_db"
echo "SELECT * FROM products;"
echo "SELECT * FROM orders;"
echo "SELECT * FROM order_items;"
echo "SELECT * FROM promo_codes;"
echo "SELECT * FROM user_operations;"

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}  Тестирование завершено!${NC}"
echo -e "${GREEN}========================================${NC}\n"