-- Миграция 1: Создание таблицы пользователей
-- Файл: 000001_create_users_table.up.sql

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индекс для быстрого поиска по email
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Добавляем тестовые данные
INSERT INTO users (name, email) VALUES
    ('Иван Иванов',   'ivan@example.com'),
    ('Мария Петрова', 'maria@example.com');
