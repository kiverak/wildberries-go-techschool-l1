package database

import (
	"l1/internal/model"
	"sync"
	"time"
)

// cacheEntry хранит заказ и время истечения
type cacheEntry struct {
	order     *model.OrderData
	expiresAt time.Time
}

type MemoryCache struct {
	cache  map[string]cacheEntry
	mu     sync.RWMutex
	ttl    time.Duration
	once   sync.Once
	stopCh chan struct{}
}

// NewMemoryCache создает новый кэш с TTL и запускает фоновую очистку
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	mc := &MemoryCache{
		cache:  make(map[string]cacheEntry),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	// Запускаем фоновую очистку каждые 5 минут
	go mc.cleanupLoop(5 * time.Minute)

	return mc
}

// Get получает значение из кэша (thread-safe, lazy invalidation)
func (m *MemoryCache) Get(uid string) (*model.OrderData, bool) {
	m.mu.RLock()
	entry, ok := m.cache[uid]
	m.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Проверяем TTL
	if time.Now().After(entry.expiresAt) {
		m.Delete(uid)
		return nil, false
	}

	return entry.order, true
}

// Set устанавливает значение в кэш (thread-safe, с TTL)
func (m *MemoryCache) Set(uid string, order *model.OrderData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[uid] = cacheEntry{
		order:     order,
		expiresAt: time.Now().Add(m.ttl),
	}
}

// Delete реализует инвалидацию кэша (thread-safe)
func (m *MemoryCache) Delete(uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, uid)
}

// Count возвращает количество элементов в кэше (thread-safe)
func (m *MemoryCache) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.cache)
}

// cleanupLoop — фоновая очистка устаревших записей
func (m *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			expiredKeys := make([]string, 0)
			// 1. Собираем ключи для удаления под блокировкой на чтение
			m.mu.RLock()
			for uid, entry := range m.cache {
				if time.Now().After(entry.expiresAt) {
					expiredKeys = append(expiredKeys, uid)
				}
			}
			m.mu.RUnlock()

			// 2. Если есть что удалять, блокируем для записи и удаляем
			if len(expiredKeys) > 0 {
				m.mu.Lock()
				for _, uid := range expiredKeys {
					delete(m.cache, uid)
				}
				m.mu.Unlock()
			}
		case <-m.stopCh:
			return
		}
	}
}

// Close останавливает фоновую очистку
func (m *MemoryCache) Close() {
	m.once.Do(func() { // сработает при первом вызове, повторные вызовы просто ничего не будут делать
		close(m.stopCh)
	})
}
