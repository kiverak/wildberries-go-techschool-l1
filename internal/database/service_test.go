package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"l1/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Моки ---

// MockDB — мок для OrderDB
type MockDB struct {
	mock.Mock
}

// Реализуем методы интерфейса OrderDB

func (m *MockDB) SaveOrder(ctx context.Context, order model.OrderData) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockDB) GetOrderByUID(ctx context.Context, orderUID string) (*model.OrderData, error) {
	args := m.Called(ctx, orderUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OrderData), args.Error(1)
}

func (m *MockDB) GetRecentOrderUIDs(ctx context.Context, since time.Time) ([]string, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDB) Close() {
	m.Called()
}

// MockCache — мок для OrderCache
type MockCache struct {
	mock.Mock
}

// Реализуем методы интерфейса OrderCache

func (m *MockCache) Get(uid string) (*model.OrderData, bool) {
	args := m.Called(uid)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*model.OrderData), args.Bool(1)
}

func (m *MockCache) Set(uid string, order *model.OrderData) {
	m.Called(uid, order)
}

func (m *MockCache) Delete(uid string) {
	m.Called(uid)
}

func (m *MockCache) Count() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockCache) Close() {
	m.Called()
}

// --- Тесты ---

// TestService_GetOrderByUID_CacheHit проверяет случай, когда заказ найден в кэше.
func TestService_GetOrderByUID_CacheHit(t *testing.T) {
	// --- Arrange ---
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	service := NewService(mockDB, mockCache)

	testOrder := &model.OrderData{OrderUID: "test-uid"}

	// Ожидаем вызов Get() в кэше, который вернет заказ
	mockCache.On("Get", "test-uid").Return(testOrder, true).Once()

	// --- Act ---
	order, err := service.GetOrderByUID(context.Background(), "test-uid")

	// --- Assert ---
	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, "test-uid", order.OrderUID)

	// Убедимся, что к базе данных НЕ было обращений
	mockDB.AssertNotCalled(t, "GetOrderByUID", mock.Anything, mock.Anything)
	// Убеждаемся, что Get был вызван
	mockCache.AssertExpectations(t)
}

// TestService_GetOrderByUID_CacheMiss проверяет (не найдено в кэше -> найдено в БД -> сохранено в кэш)
func TestService_GetOrderByUID_CacheMiss(t *testing.T) {
	// --- Arrange ---
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	service := NewService(mockDB, mockCache)

	testOrder := &model.OrderData{OrderUID: "test-uid"}

	// 1. Ожидаем, что не найдено в кэше
	mockCache.On("Get", "test-uid").Return(nil, false).Once()
	// 2. Ожидаем обращение к БД
	mockDB.On("GetOrderByUID", mock.Anything, "test-uid").Return(testOrder, nil).Once()
	// 3. Ожидаем запись в кэш
	mockCache.On("Set", "test-uid", testOrder).Return().Once()

	// --- Act ---
	order, err := service.GetOrderByUID(context.Background(), "test-uid")

	// --- Assert ---
	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, "test-uid", order.OrderUID)

	// Проверяем, что все ожидаемые вызовы были сделаны
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestService_GetOrderByUID_CacheMissAndDBError проверяет (не найдено в кэше -> не найдено в БД)
func TestService_GetOrderByUID_CacheMissAndDBError(t *testing.T) {
	// --- Arrange ---
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	service := NewService(mockDB, mockCache)

	dbError := errors.New("record not found")

	// 1. Ожидаем, что не найдено в кэше
	mockCache.On("Get", "test-uid").Return(nil, false).Once()
	// 2. Ожидаем обращение к БД, которое вернет ошибку
	mockDB.On("GetOrderByUID", mock.Anything, "test-uid").Return(nil, dbError).Once()

	// --- Act ---
	order, err := service.GetOrderByUID(context.Background(), "test-uid")

	// --- Assert ---
	require.Error(t, err)
	assert.Nil(t, order)
	assert.ErrorIs(t, err, dbError)

	// Убедимся, что Set в кэш НЕ вызывался
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestService_SaveOrder_Success проверяет (сохранено в БД -> сохранено в кэш)
func TestService_SaveOrder_Success(t *testing.T) {
	// --- Arrange ---
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	service := NewService(mockDB, mockCache)

	testOrder := model.OrderData{OrderUID: "test-uid"}

	// 1. Ожидаем сохранение в БД
	mockDB.On("SaveOrder", mock.Anything, testOrder).Return(nil).Once()
	// 2. Ожидаем сохранение в кэш
	mockCache.On("Set", "test-uid", &testOrder).Return().Once()

	// --- Act ---
	err := service.SaveOrder(context.Background(), testOrder)

	// --- Assert ---
	require.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestService_SaveOrder_DBError проверяет (ошибка в БД -> кэш не обновлен)
func TestService_SaveOrder_DBError(t *testing.T) {
	// --- Arrange ---
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	service := NewService(mockDB, mockCache)

	testOrder := model.OrderData{OrderUID: "test-uid"}
	dbError := errors.New("db connection failed")

	// Ожидаем сохранение в БД, которое вернет ошибку
	mockDB.On("SaveOrder", mock.Anything, testOrder).Return(dbError).Once()

	// --- Act ---
	err := service.SaveOrder(context.Background(), testOrder)

	// --- Assert ---
	require.Error(t, err)
	assert.ErrorIs(t, err, dbError)

	// Убедимся, что Set в кэш НЕ вызывался
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
	mockDB.AssertExpectations(t)
}

// TestService_Close_Simple проверяет вызов db.Close()
func TestService_Close_Simple(t *testing.T) {
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	s := &Service{db: mockDB, cache: mockCache}

	// 1. Ожидаем db.Close()
	mockDB.On("Close").Return().Once()
	// 2. MockCache тоже реализует Close(), поэтому его вызов тоже ожидается
	mockCache.On("Close").Return().Once()

	// 3. Вызываем метод
	s.Close()

	// 4. Проверяем
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestService_Close_WithCacheClose проверяет вызов db.Close() и cache.Close()
func TestService_Close_WithCacheClose(t *testing.T) {
	// --- Arrange ---
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	service := NewService(mockDB, mockCache)

	mockDB.On("Close").Return().Once()
	mockCache.On("Close").Return().Once()

	// --- Act ---
	service.Close()

	// --- Assert ---
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestService_InvalidateOrder проверяет вызов cache.Delete
func TestService_InvalidateOrder(t *testing.T) {
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	s := &Service{db: mockDB, cache: mockCache}

	uid := "delete-me"

	// 1. Ожидаем cache.Delete()
	mockCache.On("Delete", uid).Return().Once()

	// 2. Вызываем метод
	s.InvalidateOrder(uid)

	// 3. Проверяем
	mockCache.AssertExpectations(t)
}

// TestService_warmUpCache_Success (тестируем неэкспортируемый метод)
func TestService_warmUpCache_Success(t *testing.T) {
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	s := &Service{db: mockDB, cache: mockCache}

	order1 := &model.OrderData{OrderUID: "uid1"}
	order2 := &model.OrderData{OrderUID: "uid2"}
	uids := []string{"uid1", "uid2"}

	// 1. Ожидаем GetRecentOrderUIDs
	mockDB.On("GetRecentOrderUIDs", mock.Anything, mock.Anything).Return(uids, nil).Once()

	// 2. Ожидаем GetOrderByUID для каждого uid
	mockDB.On("GetOrderByUID", mock.Anything, "uid1").Return(order1, nil).Once()
	mockDB.On("GetOrderByUID", mock.Anything, "uid2").Return(order2, nil).Once()

	// 3. Ожидаем Set для каждого заказа
	mockCache.On("Set", "uid1", order1).Return().Once()
	mockCache.On("Set", "uid2", order2).Return().Once()

	// 4. Вызываем метод
	err := s.warmUpCache(context.Background(), time.Time{}) // time.Time{} - просто заглушка

	// 5. Проверяем
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// TestService_warmUpCache_PartialFail (ошибка при загрузке одного из заказов)
func TestService_warmUpCache_PartialFail(t *testing.T) {
	mockDB := new(MockDB)
	mockCache := new(MockCache)
	s := &Service{db: mockDB, cache: mockCache}

	order2 := &model.OrderData{OrderUID: "uid2"}
	uids := []string{"uid1-fail", "uid2"}
	dbError := errors.New("failed to get uid1")

	// 1. Ожидаем GetRecentOrderUIDs
	mockDB.On("GetRecentOrderUIDs", mock.Anything, mock.Anything).Return(uids, nil).Once()

	// 2. Ожидаем GetOrderByUID для uid1 (вернет ошибку)
	mockDB.On("GetOrderByUID", mock.Anything, "uid1-fail").Return(nil, dbError).Once()
	//    Ожидаем GetOrderByUID для uid2 (вернет успех)
	mockDB.On("GetOrderByUID", mock.Anything, "uid2").Return(order2, nil).Once()

	// 3. Ожидаем Set только для uid2
	mockCache.On("Set", "uid2", order2).Return().Once()

	// 4. Вызываем метод
	err := s.warmUpCache(context.Background(), time.Time{})

	// 5. Проверяем
	assert.NoError(t, err) // Ошибки в цикле логируются, но не прерывают прогрев
	mockDB.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	// Убеждаемся, что Set не вызывался для "uid1-fail"
	mockCache.AssertNotCalled(t, "Set", "uid1-fail", mock.Anything)
}
