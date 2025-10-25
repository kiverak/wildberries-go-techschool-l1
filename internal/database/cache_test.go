package database

import (
	"fmt"
	"l1/internal/model"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// newTestOrder создает мок-заказ для тестов.
func newTestOrder(uid string) *model.OrderData {
	return &model.OrderData{
		OrderUID: uid,
	}
}

// TestMemoryCache_SetAndGet — базовый тест на установку и получение.
func TestMemoryCache_SetAndGet(t *testing.T) {
	ttl := 5 * time.Minute
	cache := NewMemoryCache(ttl)
	defer cache.Close()

	order := newTestOrder("order-123")
	cache.Set("order-123", order)

	retrieved, ok := cache.Get("order-123")
	if !ok {
		t.Fatal("Не удалось получить заказ, который был только что установлен")
	}
	if retrieved == nil {
		t.Fatal("Получен nil вместо заказа")
	}
	if retrieved.OrderUID != "order-123" {
		t.Errorf("Получен заказ с неверным UID. Ожидалось 'order-123', получено %q", retrieved.OrderUID)
	}
}

// TestMemoryCache_GetNotFound — тест получения несуществующего ключа.
func TestMemoryCache_GetNotFound(t *testing.T) {
	cache := NewMemoryCache(5 * time.Minute)
	defer cache.Close()

	retrieved, ok := cache.Get("non-existent-key")
	if ok {
		t.Error("Получено 'true' для несуществующего ключа")
	}
	if retrieved != nil {
		t.Error("Получен заказ вместо nil для несуществующего ключа")
	}
}

// TestMemoryCache_LazyTTLExpiration — тест "ленивой" инвалидации при Get.
func TestMemoryCache_LazyTTLExpiration(t *testing.T) {
	// Используем очень маленький TTL для теста
	ttl := 50 * time.Millisecond
	cache := NewMemoryCache(ttl)
	defer cache.Close()

	order := &model.OrderData{OrderUID: "test-uid-ttl"}
	cache.Set("test-uid-ttl", order)

	// 1. Сразу после добавления элемент должен быть в кэше
	_, ok := cache.Get("test-uid-ttl")
	assert.True(t, ok, "Элемент должен быть в кэше сразу после добавления")

	// 2. Ждем, пока TTL истечет
	time.Sleep(ttl + 10*time.Millisecond)

	// 3. Теперь элемента не должно быть в кэше (ленивая инвалидация)
	_, ok = cache.Get("test-uid-ttl")
	assert.False(t, ok, "Элемент должен был быть удален по TTL")
	assert.Equal(t, 0, cache.Count(), "Кэш должен быть пуст после истечения TTL и Get")
}

// TestMemoryCache_GetBeforeTTL — тест получения до истечения TTL.
func TestMemoryCache_GetBeforeTTL(t *testing.T) {
	ttl := 50 * time.Millisecond
	cache := NewMemoryCache(ttl)
	defer cache.Close()

	cache.Set("order-1", newTestOrder("order-1"))

	// Ждем меньше, чем TTL
	time.Sleep(ttl - 30*time.Millisecond)

	retrieved, ok := cache.Get("order-1")
	if !ok {
		t.Error("Получено 'false' для ключа с активным TTL")
	}
	if retrieved == nil {
		t.Error("Получен nil вместо заказа для ключа с активным TTL")
	}
}

// TestMemoryCache_Delete — тест удаления из кэша.
func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(5 * time.Minute)
	defer cache.Close()

	cache.Set("order-1", newTestOrder("order-1"))
	_, ok := cache.Get("order-1")
	if !ok {
		t.Fatal("Не удалось получить заказ для удаления")
	}

	cache.Delete("order-1")
	retrieved, ok := cache.Get("order-1")
	if ok {
		t.Error("Получено 'true' для удаленного ключа")
	}
	if retrieved != nil {
		t.Error("Получен заказ вместо nil для удаленного ключа")
	}
}

// TestMemoryCache_CleanupLoop - тест фоновой очистки кэша.
func TestMemoryCache_CleanupLoop(t *testing.T) {
	// TTL 50мс, интервал очистки 20мс
	ttl := 50 * time.Millisecond
	cleanupInterval := 20 * time.Millisecond

	cache := NewMemoryCache(ttl)
	// Переопределяем интервал очистки для теста
	cache.stopCh = make(chan struct{})
	go cache.cleanupLoop(cleanupInterval)
	defer cache.Close()

	cache.Set("test-uid-cleanup", &model.OrderData{OrderUID: "test-uid-cleanup"})
	assert.Equal(t, 1, cache.Count())

	// Ждем дольше, чем TTL и интервал очистки
	time.Sleep(ttl + cleanupInterval)

	// Проверяем, что фоновая очистка сработала
	cache.mu.RLock()
	count := len(cache.cache)
	cache.mu.RUnlock()

	assert.Equal(t, 0, count, "Фоновая очистка должна была удалить просроченный элемент")
}

// TestMemoryCache_Count — тест подсчета элементов.
func TestMemoryCache_Count(t *testing.T) {
	cache := NewMemoryCache(1 * time.Minute)
	defer cache.Close()

	if cache.Count() != 0 {
		t.Errorf("Ожидалось 0 элементов, получено %d", cache.Count())
	}

	cache.Set("order-1", newTestOrder("order-1"))
	if cache.Count() != 1 {
		t.Errorf("Ожидался 1 элемент, получено %d", cache.Count())
	}

	cache.Set("order-2", newTestOrder("order-2"))
	if cache.Count() != 2 {
		t.Errorf("Ожидалось 2 элемента, получено %d", cache.Count())
	}

	cache.Delete("order-1")
	if cache.Count() != 1 {
		t.Errorf("Ожидался 1 элемент после удаления, получено %d", cache.Count())
	}
}

// TestMemoryCache_Close — тест остановки фоновой горутины.
func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache(1 * time.Second)
	// Просто вызываем Close() - тест пройдет, если не будет паники.
	cache.Close()
	// Повторный вызов Close() на уже закрытом канале также не должен паниковать.
	cache.Close()
}

// TestMemoryCache_Concurrency — тест на "гонку" (race condition).
func TestMemoryCache_Concurrency(t *testing.T) {
	cache := NewMemoryCache(100 * time.Millisecond)
	defer cache.Close()

	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines * 2)

	// 100 горутин на запись
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			defer wg.Done()
			uid := fmt.Sprintf("order-%d", j)
			cache.Set(uid, newTestOrder(uid))
		}(i)
	}

	// 100 горутин на чтение
	for i := 0; i < numGoroutines; i++ {
		go func(j int) {
			defer wg.Done()
			uid := fmt.Sprintf("order-%d", j)
			// Просто читаем, результат не важен, главное - отсутствие паники
			cache.Get(uid)
		}(i)
	}

	wg.Wait()

	// Если мы дошли сюда без паники от 'go test -race', тест пройден.
	// Мы не можем точно предсказать Count() из-за TTL,
	// но можем проверить, что он не паникует.
	t.Logf("Финальное количество в кэше после теста на конкурентность: %d", cache.Count())
}
