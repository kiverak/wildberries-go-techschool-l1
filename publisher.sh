#!/bin/bash

# --- Конфигурация ---
KAFKA_CONTAINER="kafka1"
KAFKA_TOPIC="orders"
KAFKA_INTERNAL_BOOTSTRAP_SERVER="kafka1:29092" # Для команд внутри Docker
JSON_TEMPLATE_FILE="./internal/test/test_model.json"
MESSAGE_INTERVAL_SECONDS=2 # Интервал между сообщениями

# --- Пути к утилитам Kafka внутри контейнера ---
KAFKA_TOPICS_CMD="//opt/bitnami/kafka/bin/kafka-topics.sh"
KAFKA_PRODUCER_CMD="//opt/bitnami/kafka/bin/kafka-console-producer.sh"

# --- Создаём топик ---
create_topic_if_not_exists() {
    # Проверяем, есть ли топик в списке
    echo "Проверка существования топика '$KAFKA_TOPIC'..."
    if docker exec "$KAFKA_CONTAINER" "$KAFKA_TOPICS_CMD" --bootstrap-server "$KAFKA_INTERNAL_BOOTSTRAP_SERVER" --list | grep -q "^${KAFKA_TOPIC}$"; then
        echo "Топик '$KAFKA_TOPIC' уже существует."
    else  # создаём топик только если его нет
        echo "Топик '$KAFKA_TOPIC' не найден. Создание..."
        docker exec "$KAFKA_CONTAINER" "$KAFKA_TOPICS_CMD" \
            --create \
            --topic "$KAFKA_TOPIC" \
            --bootstrap-server "$KAFKA_INTERNAL_BOOTSTRAP_SERVER" \
            --partitions 3 \
            --replication-factor 3
        echo "Топик '$KAFKA_TOPIC' успешно создан."
    fi
}

# запускаем функцию для создания топика
create_topic_if_not_exists

echo -e "\nНачинаю отправку сообщений в топик '$KAFKA_TOPIC'. Нажмите Ctrl+C для остановки."

# Функция бесконечно отправляет сообщения, пока не будет остановлена через Ctrl+C
while true; do
    # Генерируем уникальный UID на основе времени и случайной строки
    UNIQUE_ID="order-$(date +%s)-$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 8)"

    # Читаем шаблон, подставляем новый UID и отправляем в Kafka
    jq -c --arg uid "$UNIQUE_ID" '(.order_uid = $uid) | (.payment.transaction = $uid)' "$JSON_TEMPLATE_FILE" | \
    docker exec -i "$KAFKA_CONTAINER" "$KAFKA_PRODUCER_CMD" --topic "$KAFKA_TOPIC" --bootstrap-server "$KAFKA_INTERNAL_BOOTSTRAP_SERVER"

    echo "Отправлено сообщение с order_uid: $UNIQUE_ID"
    sleep "$MESSAGE_INTERVAL_SECONDS"
done