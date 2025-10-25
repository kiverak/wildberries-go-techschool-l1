package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"l1/internal/model"

	"github.com/segmentio/kafka-go"
)

// OrderSaver определяет интерфейс для сохранения заказа.
type OrderSaver interface {
	SaveOrder(ctx context.Context, order model.OrderData) error
}

// handleMessage распаковывает, валидирует и сохраняет заказ в хранилище.
func handleMessage(ctx context.Context, msgValue []byte, store OrderSaver) error {
	var orderMsg model.OrderData
	if err := json.Unmarshal(msgValue, &orderMsg); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if err := orderMsg.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации заказа (%s): %w", orderMsg.OrderUID, err)
	}

	if err := store.SaveOrder(ctx, orderMsg); err != nil {
		return fmt.Errorf("ошибка сохранения заказа в БД (%s): %w", orderMsg.OrderUID, err)
	}

	log.Printf("Заказ %s успешно сохранен", orderMsg.OrderUID)
	return nil
}

// Start запускает consumer'а Kafka.
func Start(ctx context.Context, brokers []string, topic string, store OrderSaver) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        "order-processor-group", // Группа консьюмеров
		MinBytes:       10e3,                    // 10KB
		MaxBytes:       10e6,                    // 10MB
		CommitInterval: time.Second,             // Фиксируем смещение каждую секунду
	})
	defer func(r *kafka.Reader) {
		err := r.Close()
		if err != nil {
			log.Printf("ошибка закрытия Reader: %v", err)
		}
	}(r)

	log.Printf("Запущен consumer для топика '%s'", topic)

	for {
		// Читаем сообщение из Kafka
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			// Если контекст отменен, выходим из цикла
			if ctx.Err() != nil {
				break
			}
			log.Printf("Ошибка при чтении сообщения: %v", err)
			continue
		}

		log.Printf("Получено сообщение: offset=%d, key=%s, value=%s\n", msg.Offset, string(msg.Key), string(msg.Value))

		if err := handleMessage(ctx, msg.Value, store); err != nil {
			log.Printf("ошибка обработки сообщения: %v", err)
		}
	}

	log.Println("Consumer остановлен.")
}
