package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	store, err := database.NewStore(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.DB.Close()

	// Запускаем Kafka consumer в отдельной горутине
	go consumer.Start(ctx, cfg.KafkaBrokers, cfg.KafkaTopic, store)

	// Запускаем веб-сервер
	webServer := server.New(store)
	webServer.Start(cfg.ServerAddr)
}
