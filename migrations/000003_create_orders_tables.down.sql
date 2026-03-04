-- Удаление триггера
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;

-- Удаление индексов user_operations
DROP INDEX IF EXISTS idx_user_operations_created_at;
DROP INDEX IF EXISTS idx_user_operations_type;
DROP INDEX IF EXISTS idx_user_operations_user_id;

-- Удаление индексов promo_codes
DROP INDEX IF EXISTS idx_promo_codes_active;
DROP INDEX IF EXISTS idx_promo_codes_code;

-- Удаление индексов order_items
DROP INDEX IF EXISTS idx_order_items_product_id;
DROP INDEX IF EXISTS idx_order_items_order_id;

-- Удаление индексов orders
DROP INDEX IF EXISTS idx_orders_promo_code_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_user_id;

-- Удаление таблиц
DROP TABLE IF EXISTS user_operations;
DROP TABLE IF EXISTS promo_codes;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;