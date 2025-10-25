package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"l1/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Мок ---

// MockOrderGetter — мок для интерфейса OrderGetter.
type MockOrderGetter struct {
	mock.Mock
}

func (m *MockOrderGetter) GetOrderByUID(ctx context.Context, orderUID string) (*model.OrderData, error) {
	args := m.Called(ctx, orderUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OrderData), args.Error(1)
}

// --- Тесты ---

// TestHandleGetOrder_Success - тест успешного запроса
func TestHandleGetOrder_Success(t *testing.T) {
	// --- Arrange ---
	// Создаем мок и настраиваем его поведение
	mockStore := new(MockOrderGetter)
	testOrder := &model.OrderData{OrderUID: "test-uid-123", TrackNumber: "WBILMTESTTRACK"}
	mockStore.On("GetOrderByUID", mock.Anything, "test-uid-123").Return(testOrder, nil).Once()

	// Создаем сервер с моком
	server := New(mockStore)

	// Создаем фейковый HTTP-запрос
	req := httptest.NewRequest(http.MethodGet, "/order/test-uid-123", nil)
	rr := httptest.NewRecorder() // Это "перехватчик" ответа

	// --- Act ---
	server.handleGetOrder(rr, req)

	// --- Assert ---
	assert.Equal(t, http.StatusOK, rr.Code, "Код ответа должен быть 200 OK")

	var responseOrder model.OrderData
	err := json.Unmarshal(rr.Body.Bytes(), &responseOrder)
	require.NoError(t, err, "Тело ответа должно быть валидным JSON")
	assert.Equal(t, testOrder.OrderUID, responseOrder.OrderUID, "UID заказа в ответе не совпадает")
	assert.Equal(t, testOrder.TrackNumber, responseOrder.TrackNumber, "TrackNumber в ответе не совпадает")

	mockStore.AssertExpectations(t)
}

// TestHandleGetOrder_NotFound - тест ошибки "не найдено"
func TestHandleGetOrder_NotFound(t *testing.T) {
	// --- Arrange ---
	mockStore := new(MockOrderGetter)
	mockStore.On("GetOrderByUID", mock.Anything, "not-found-uid").Return(nil, errors.New("not found")).Once()

	server := New(mockStore)

	req := httptest.NewRequest(http.MethodGet, "/order/not-found-uid", nil)
	rr := httptest.NewRecorder()

	// --- Act ---
	server.handleGetOrder(rr, req)

	// --- Assert ---
	assert.Equal(t, http.StatusNotFound, rr.Code, "Код ответа должен быть 404 Not Found")
	assert.Contains(t, rr.Body.String(), "Заказ не найден", "Тело ответа должно содержать сообщение об ошибке")

	mockStore.AssertExpectations(t)
}

// TestHandleGetOrder_BadRequest_NoUID - тест ошибки "не указан UID"
func TestHandleGetOrder_BadRequest_NoUID(t *testing.T) {
	// --- Arrange ---
	mockStore := new(MockOrderGetter)
	server := New(mockStore)

	req := httptest.NewRequest(http.MethodGet, "/order/", nil) // Пустой UID
	rr := httptest.NewRecorder()

	// --- Act ---
	server.handleGetOrder(rr, req)

	// --- Assert ---
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Код ответа должен быть 400 Bad Request")
	assert.Contains(t, rr.Body.String(), "Не указан UID заказа", "Тело ответа должно содержать сообщение об ошибке")

	// Убедимся, что до GetOrderByUID не вызывался
	mockStore.AssertNotCalled(t, "GetOrderByUID", mock.Anything, mock.Anything)
}
