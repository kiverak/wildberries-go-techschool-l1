## Локальная разработка

**Устанавливаем необходимые библиотеки**

**Клиент для Kafka**

`go get github.com/segmentio/kafka-go`

**Драйвер для PostgreSQL**

`go get github.com/jackc/pgx/v5`

**Для загрузки переменных окружения (опционально)**

`go get github.com/joho/godotenv`

**Запуск контейнера Postgres**

`docker-compose up -d`

**Запуск контейнера Kafka**

`docker-compose -f docker-compose-kafka.yml up -d`

Теперь локально работает кластер Kafka из трех брокеров. Можно подключаться к нему, используя bootstrap-серверы: localhost:9092,localhost:9094,localhost:9096

**Создать топик в Kafka**

`docker exec -it kafka1 //opt/bitnami/kafka/bin/kafka-topics.sh --create --topic orders --bootstrap-server kafka1:29092 --partitions 3 --replication-factor 3`

**Запустить Go-приложение**

`go run cmd/main.go`

**Отправьте тестовое сообщение в Kafka**

`docker exec -it kafka1 /opt/bitnami/kafka/bin/kafka-console-producer.sh --topic orders --bootstrap-server localhost:9092 {JSON_MESSAGE_HERE}`

или

`cat ./internal/test/test_model.json | tr -d '\n\r' | docker exec -i kafka1 //opt/bitnami/kafka/bin/kafka-console-producer.sh --topic orders --bootstrap-server kafka1:29092`
