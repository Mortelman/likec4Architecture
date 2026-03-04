-- Откат миграции 1: Удаление таблицы пользователей
-- Файл: 000001_create_users_table.down.sql

DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
