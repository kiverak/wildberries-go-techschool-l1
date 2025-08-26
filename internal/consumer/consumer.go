package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"l1/internal/database"
	"l1/internal/model"

	"github.com/segmentio/kafka-go"
)

// Start запускает consumer'а Kafka.
func Start(ctx context.Context, brokers []string, topic string, store *database.Store) {
	// Настраиваем Kafka Reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        "order-processor-group", // Группа консьюмеров
		MinBytes:       10e3,                    // 10KB
		MaxBytes:       10e6,                    // 10MB
		CommitInterval: time.Second,             // Фиксируем смещение каждую секунду
	})
	defer r.Close()

	log.Printf("Запущен consumer для топика '%s'", topic)

	for {
		// Читаем сообщение из Kafka
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			// Если контекст отменен, выходим из цикла
			if ctx.Err() != nil {
				break
			}
			log.Printf("ошибка при чтении сообщения: %v", err)
			continue
		}

		log.Printf("Получено сообщение: offset=%d, key=%s, value=%s\n", msg.Offset, string(msg.Key), string(msg.Value))

		// Парсим JSON
		var orderMsg model.OrderData
		if err := json.Unmarshal(msg.Value, &orderMsg); err != nil {
			log.Printf("ошибка парсинга JSON: %v", err)
			continue
		}

		// Сохраняем заказ в базу данных
		if err := store.SaveOrder(ctx, orderMsg); err != nil {
			log.Printf("ошибка сохранения заказа в БД (%s): %v", orderMsg.OrderUID, err)
		} else {
			log.Printf("Заказ %s успешно сохранен", orderMsg.OrderUID)
		}
	}

	log.Println("Consumer остановлен.")
}
