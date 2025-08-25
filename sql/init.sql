-- Отключаем проверки внешних ключей на время создания таблиц для удобства
SET session_replication_role = 'replica';

-- Таблица для хранения информации о доставке
CREATE TABLE IF NOT EXISTS delivery (
    order_uid VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    zip VARCHAR(50) NOT NULL,
    city VARCHAR(100) NOT NULL,
    address VARCHAR(255) NOT NULL,
    region VARCHAR(100),
    email VARCHAR(100) NOT NULL
    );

-- Таблица для хранения информации об оплате
CREATE TABLE IF NOT EXISTS payment (
    transaction_id VARCHAR(255) PRIMARY KEY,
    request_id VARCHAR(255),
    currency VARCHAR(10) NOT NULL,
    provider VARCHAR(50),
    amount INT NOT NULL,
    payment_dt BIGINT NOT NULL,
    bank VARCHAR(100),
    delivery_cost INT NOT NULL,
    goods_total INT NOT NULL,
    custom_fee INT
    );

-- Таблица для хранения товаров в заказе
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY, -- собственный первичный ключ для удобства
    order_uid VARCHAR(255) NOT NULL,
    chrt_id INT NOT NULL,
    track_number VARCHAR(255) NOT NULL,
    price INT NOT NULL,
    rid VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sale INT,
    size VARCHAR(10),
    total_price INT NOT NULL,
    nm_id INT NOT NULL,
    brand VARCHAR(100),
    status INT
    );

-- Основная таблица заказов
CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(255) PRIMARY KEY,
    track_number VARCHAR(255) NOT NULL,
    entry VARCHAR(50),
    customer_id VARCHAR(255) NOT NULL,
    delivery_service VARCHAR(100),
    shardkey VARCHAR(10),
    sm_id INT,
    date_created TIMESTAMP WITH TIME ZONE NOT NULL,
    oof_shard VARCHAR(10),
    locale VARCHAR(10),
    internal_signature VARCHAR(255),
    -- Связываем с другими таблицами через внешние ключи
    CONSTRAINT fk_delivery FOREIGN KEY (order_uid) REFERENCES delivery(order_uid) ON DELETE CASCADE,
    CONSTRAINT fk_payment FOREIGN KEY (order_uid) REFERENCES payment(transaction_id) ON DELETE CASCADE
    );

-- Добавляем внешний ключ для items после создания orders
ALTER TABLE items
    ADD CONSTRAINT fk_items_order FOREIGN KEY (order_uid) REFERENCES orders(order_uid) ON DELETE CASCADE;

-- Создаем индексы для ускорения поиска
CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_items_order_uid_chrt_id ON items(order_uid, chrt_id);

-- Включаем проверки обратно
SET session_replication_role = 'origin';