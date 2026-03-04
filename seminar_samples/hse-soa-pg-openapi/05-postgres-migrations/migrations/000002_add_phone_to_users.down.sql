-- Откат миграции 2: Удаление поля телефона
-- Файл: 000002_add_phone_to_users.down.sql

ALTER TABLE users
    DROP COLUMN IF EXISTS phone;
