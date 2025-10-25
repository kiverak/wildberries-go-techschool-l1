package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"l1/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockPostgresStore создает PostgresStore с моком для тестов.
func newMockPostgresStore(t *testing.T) (*PostgresStore, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	store := &PostgresStore{DB: mock}
	return store, mock
}

// newTestOrderData создает полный объект заказа для использования в тестах.
func newTestOrderData(uid string) model.OrderData {
	return model.OrderData{
		OrderUID:    uid,
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: model.Payment{
			Transaction:  uid,
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []model.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "TRACK123",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389232,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
	}
}

// TestPostgresStore_SaveOrder_Success проверяет успешное сохранение заказа
func TestPostgresStore_SaveOrder_Success(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	order := newTestOrderData("order-success")

	// 1. Ожидаем начала транзакции
	mock.ExpectBegin()

	// 2. Ожидаем INSERT в delivery
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO delivery`)).
		WithArgs(
			order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
			order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// 3. Ожидаем INSERT в payment
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO payment`)).
		WithArgs(
			order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
			order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
			order.Payment.GoodsTotal, order.Payment.CustomFee,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// 4. Ожидаем INSERT в orders
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO orders`)).
		WithArgs(
			order.OrderUID, order.TrackNumber, order.Entry, order.CustomerID, order.DeliveryService,
			order.Shardkey, order.SmID, order.DateCreated, order.OofShard, order.Locale, order.InternalSignature,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// 5. Ожидаем INSERT для каждого товара (в нашем случае 1)
	for _, item := range order.Items {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO items`)).
			WithArgs(
				order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
				item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	// 6. Ожидаем Commit транзакции
	mock.ExpectCommit()

	// Вызываем тестируемую функцию
	err := store.SaveOrder(ctx, order)
	assert.NoError(t, err, "error was not expected while saving order")

	// Проверяем, что все ожидания были выполнены
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestPostgresStore_SaveOrder_FailDelivery проверяет откат транзакции при ошибке
func TestPostgresStore_SaveOrder_FailDelivery(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	order := newTestOrderData("order-fail")
	dbErr := errors.New("db error on delivery")

	// 1. Ожидаем начала транзакции
	mock.ExpectBegin()

	// 2. Ожидаем INSERT в delivery, который вернет ошибку
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO delivery`)).
		WithArgs(
			order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
			order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
		).
		WillReturnError(dbErr)

	// 3. Ожидаем Rollback транзакции
	mock.ExpectRollback()

	// Вызываем функцию
	err := store.SaveOrder(ctx, order)
	require.Error(t, err, "expected an error, but got nil")
	assert.ErrorIs(t, err, dbErr, "expected error to wrap the db error")

	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestPostgresStore_SaveOrder_FailCommit проверяет ошибку на этапе Commit
func TestPostgresStore_SaveOrder_FailCommit(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	order := newTestOrderData("order-fail-commit")
	commitErr := errors.New("commit error")

	mock.ExpectBegin()
	// (Все Exec'и проходят успешно)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO delivery`)).
		WithArgs(
			order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
			order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO payment`)).
		WithArgs(
			order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
			order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
			order.Payment.GoodsTotal, order.Payment.CustomFee,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO orders`)).
		WithArgs(
			order.OrderUID, order.TrackNumber, order.Entry, order.CustomerID, order.DeliveryService,
			order.Shardkey, order.SmID, order.DateCreated, order.OofShard, order.Locale, order.InternalSignature,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO items`)).
		WithArgs(
			order.OrderUID, order.Items[0].ChrtID, order.Items[0].TrackNumber, order.Items[0].Price, order.Items[0].Rid, order.Items[0].Name,
			order.Items[0].Sale, order.Items[0].Size, order.Items[0].TotalPrice, order.Items[0].NmID, order.Items[0].Brand, order.Items[0].Status,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Ожидаем Commit, который вернет ошибку
	mock.ExpectCommit().WillReturnError(commitErr)

	// Вызываем функцию
	err := store.SaveOrder(ctx, order)
	require.Error(t, err, "expected an error on commit, but got nil")
	assert.ErrorIs(t, err, commitErr, "expected error to be the commit error")

	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestPostgresStore_GetOrderByUID_Success проверяет успешное получение заказа
func TestPostgresStore_GetOrderByUID_Success(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	uid := "order-get-success"
	order := newTestOrderData(uid)
	item := order.Items[0]

	// 1. Ожидаем запрос в 'orders'
	orderRows := pgxmock.NewRows([]string{
		"order_uid", "track_number", "entry", "customer_id", "delivery_service",
		"shardkey", "sm_id", "date_created", "oof_shard", "locale", "internal_signature",
	}).AddRow(
		order.OrderUID, order.TrackNumber, order.Entry, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard, order.Locale, order.InternalSignature,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_uid, track_number, entry, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, locale, internal_signature FROM orders WHERE order_uid = $1`)).
		WithArgs(uid).
		WillReturnRows(orderRows)

	// 2. Ожидаем запрос в 'delivery'
	deliveryRows := pgxmock.NewRows([]string{
		"name", "phone", "zip", "city", "address", "region", "email",
	}).AddRow(
		order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, phone, zip, city, address, region, email FROM delivery WHERE order_uid = $1`)).
		WithArgs(uid).
		WillReturnRows(deliveryRows)

	// 3. Ожидаем запрос в 'payment'
	paymentRows := pgxmock.NewRows([]string{
		"transaction_id", "request_id", "currency", "provider", "amount",
		"payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee",
	}).AddRow(
		order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT transaction_id, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment WHERE transaction_id = $1`)).
		WithArgs(uid).
		WillReturnRows(paymentRows)

	// 4. Ожидаем запрос в 'items'
	itemsRows := pgxmock.NewRows([]string{
		"chrt_id", "track_number", "price", "rid", "name", "sale",
		"size", "total_price", "nm_id", "brand", "status",
	}).AddRow(
		item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale,
		item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_uid = $1`)).
		WithArgs(uid).
		WillReturnRows(itemsRows)

	// Вызываем функцию
	retrievedOrder, err := store.GetOrderByUID(ctx, uid)
	require.NoError(t, err, "error was not expected while getting order")
	require.NotNil(t, retrievedOrder, "retrieved order is nil")

	// Сравним ключевые поля для уверенности
	assert.Equal(t, uid, retrievedOrder.OrderUID)
	assert.Equal(t, order.Delivery.Name, retrievedOrder.Delivery.Name)
	require.Len(t, retrievedOrder.Items, 1, "wrong number of items returned")
	assert.Equal(t, item.ChrtID, retrievedOrder.Items[0].ChrtID)

	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestPostgresStore_GetOrderByUID_OrderNotFound проверяет ошибку "не найдено"
func TestPostgresStore_GetOrderByUID_OrderNotFound(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	uid := "order-not-found"

	// 1. Ожидаем запрос в 'orders', который вернет pgx.ErrNoRows
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_uid, track_number, entry, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, locale, internal_signature FROM orders WHERE order_uid = $1`)).
		WithArgs(uid).
		WillReturnError(pgx.ErrNoRows)

	// Вызываем функцию
	retrievedOrder, err := store.GetOrderByUID(ctx, uid)
	require.Error(t, err, "expected an error, but got nil")
	assert.Nil(t, retrievedOrder, "order should be nil on error")

	// Проверяем, что функция вернула обернутую ошибку pgx.ErrNoRows
	assert.ErrorIs(t, err, pgx.ErrNoRows, "expected error to wrap pgx.ErrNoRows")
	assert.Contains(t, err.Error(), fmt.Sprintf("заказ с UID %s не найден", uid), "wrong error message")

	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestPostgresStore_GetRecentOrderUIDs_Success проверяет успешное получение UID'ов
func TestPostgresStore_GetRecentOrderUIDs_Success(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	since := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	// 1. Ожидаем запрос
	rows := pgxmock.NewRows([]string{"order_uid"}).
		AddRow("uid-1").
		AddRow("uid-2").
		AddRow("uid-3")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_uid FROM orders WHERE date_created >= $1`)).
		WithArgs(since).
		WillReturnRows(rows)

	// Вызываем функцию
	uids, err := store.GetRecentOrderUIDs(ctx, since)
	require.NoError(t, err, "error was not expected while getting recent UIDs")
	require.Len(t, uids, 3, "expected 3 UIDs")
	assert.Equal(t, []string{"uid-1", "uid-2", "uid-3"}, uids, "wrong UIDs returned")

	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestPostgresStore_GetRecentOrderUIDs_QueryFail проверяет ошибку при запросе UID'ов
func TestPostgresStore_GetRecentOrderUIDs_QueryFail(t *testing.T) {
	ctx := context.Background()
	store, mock := newMockPostgresStore(t)
	defer mock.Close()
	since := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	dbErr := errors.New("query failed")

	// 1. Ожидаем запрос, который вернет ошибку
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_uid FROM orders WHERE date_created >= $1`)).
		WithArgs(since).
		WillReturnError(dbErr)

	// Вызываем функцию
	uids, err := store.GetRecentOrderUIDs(ctx, since)
	require.Error(t, err, "expected an error, but got nil")
	assert.Nil(t, uids, "UIDs should be nil on error")
	assert.ErrorIs(t, err, dbErr, "expected error to wrap the db error")

	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}
