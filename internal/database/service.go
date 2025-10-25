package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"l1/internal/model"
)

// --- 1. Интерфейсы (Контракты) ---

// OrderDB определяет контракт для работы *только* с базой данных
type OrderDB interface {
	SaveOrder(ctx context.Context, order model.OrderData) error
	GetOrderByUID(ctx context.Context, orderUID string) (*model.OrderData, error)
	GetRecentOrderUIDs(ctx context.Context, since time.Time) ([]string, error)
	Close()
}

// OrderCache определяет контракт для работы только с кэшем
type OrderCache interface {
	Get(uid string) (*model.OrderData, bool)
	Set(uid string, order *model.OrderData)
	Delete(uid string) // Метод для инвалидации
	Count() int
}

// --- 2. Сервис-Оркестратор ---

// Service — это фасад, который управляет взаимодействием между БД и кэшем
// Клиенты (например, HTTP хендлеры) работают только с ним.
type Service struct {
	db    OrderDB
	cache OrderCache
}

// NewService — конструктор, использующий Dependency Injection
func NewService(db OrderDB, cache OrderCache) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// RunBackgroundJobs запускает фоновые процессы, такие как прогрев кэша.
func (s *Service) RunBackgroundJobs(ctx context.Context) {
	go func() {
		log.Println("Запуск фонового прогрева кэша...")
		// Прогреваем данные за последние 7 дней
		since := time.Now().Add(-7 * 24 * time.Hour)
		if err := s.warmUpCache(ctx, since); err != nil {
			log.Printf("Ошибка фонового прогрева кэша: %v", err)
		} else {
			log.Printf("Фоновый прогрев кэша завершён: %d заказов", s.cache.Count())
		}
	}()
}

// warmUpCache выполняет прогрев кэша при старте
func (s *Service) warmUpCache(ctx context.Context, since time.Time) error {
	uids, err := s.db.GetRecentOrderUIDs(ctx, since)
	if err != nil {
		return fmt.Errorf("не удалось получить список заказов: %w", err)
	}

	count := 0
	for _, uid := range uids {
		// Идем напрямую в БД, чтобы загрузить данные
		order, err := s.db.GetOrderByUID(ctx, uid)
		if err != nil {
			log.Printf("Пропускаем заказ %s при прогреве: %v", uid, err)
			continue
		}
		// Напрямую кладем в кэш
		s.cache.Set(uid, order)
		count++
	}
	log.Printf("Прогрузили в кэш %d заказов", count)
	return nil
}

// SaveOrder реализует паттерн "Write-Through Cache"
func (s *Service) SaveOrder(ctx context.Context, order model.OrderData) error {
	// 1. Сначала в постоянное хранилище (БД)
	if err := s.db.SaveOrder(ctx, order); err != nil {
		return fmt.Errorf("ошибка сохранения заказа в БД: %w", err)
	}

	// 2. Затем обновляем кэш
	log.Printf("Заказ %s сохранен в БД, обновляем кэш...", order.OrderUID)
	s.cache.Set(order.OrderUID, &order)

	return nil
}

// GetOrderByUID реализует паттерн "Cache-Aside"
func (s *Service) GetOrderByUID(ctx context.Context, orderUID string) (*model.OrderData, error) {
	// 1. Пытаемся прочитать из кэша
	if order, ok := s.cache.Get(orderUID); ok {
		log.Printf("Заказ %s найден в кэше", orderUID)
		return order, nil
	}

	// 2. Если в кэше нет — читаем из базы
	log.Printf("Заказ %s не найден в кэше, обращаемся к БД...", orderUID)
	order, err := s.db.GetOrderByUID(ctx, orderUID)
	if err != nil {
		return nil, err // Ошибка (включая "не найдено")
	}

	// 3. Кладём в кэш
	s.cache.Set(orderUID, order)

	return order, nil
}

// InvalidateOrder — метод для инвалидации кэша
func (s *Service) InvalidateOrder(uid string) {
	log.Printf("Инвалидация кэша для заказа %s", uid)
	s.cache.Delete(uid)
}

// Close закрывает пулы соединений
func (s *Service) Close() {
	if s.db != nil {
		s.db.Close()
	}

	if closer, ok := s.cache.(interface{ Close() }); ok {
		closer.Close()
	}
}
