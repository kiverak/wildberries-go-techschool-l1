package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"l1/internal/config"
	"l1/internal/consumer"
	"l1/internal/database"
	"l1/internal/server"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Создаем контекст, который будет отменен при получении сигнала прерывания
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopChan
		log.Println("Получен сигнал завершения. Завершение работы...")
		cancel()
	}()

	// Подключаемся к базе данных
	dbStore, err := database.NewPostgresStore(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	// Создаем слой для работы с кэшем (in-memory)
	memCache := database.NewMemoryCache(1 * time.Hour) // инвалидация кеша через TTL 1 час

	// Создаем основной сервис, передавая ему зависимости (БД и кэш)
	orderService := database.NewService(dbStore, memCache)
	defer orderService.Close() //  закрываем соединение с БД и кеш

	// Запускаем Kafka consumer в отдельной горутине
	go consumer.Start(ctx, cfg.KafkaBrokers, cfg.KafkaTopic, orderService)

	// Запускаем веб-сервер
	webServer := server.New(orderService)
	if err := webServer.Start(cfg.ServerAddr); err != nil {
		log.Printf("Ошибка сервера: %v", err)
	}

	<-ctx.Done()
	log.Println("Приложение успешно завершило работу.")
}
