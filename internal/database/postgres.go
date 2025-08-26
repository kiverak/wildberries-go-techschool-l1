package database

import (
	"context"
	"fmt"
	"log"
	"sync"

	"l1/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	DB    *pgxpool.Pool
	cache map[string]*model.OrderData
	mu    sync.RWMutex
}

func NewStore(connString string) (*Store, error) {
	dbpool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}
	if err = dbpool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("не удалось проверить соединение с базой данных: %w", err)
	}
	log.Println("Успешное подключение к PostgreSQL!")

	store := &Store{
		DB:    dbpool,
		cache: make(map[string]*model.OrderData),
	}

	// Фоновый прогрев кэша
	go func() {
		if err := store.loadCache(context.Background()); err != nil {
			log.Printf("ошибка фонового прогрева кэша: %v", err)
		} else {
			log.Printf("Фоновый прогрев кэша завершён: %d заказов", len(store.cache))
		}
	}()

	return store, nil
}

// загружает в кэш заказы за последние 7 дней
func (s *Store) loadCache(ctx context.Context) error {
	rows, err := s.DB.Query(ctx, `
		SELECT order_uid 
		FROM orders 
		WHERE date_created >= NOW() - interval '7 days'
		`)
	if err != nil {
		return fmt.Errorf("не удалось получить список заказов: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return fmt.Errorf("ошибка при чтении uid: %w", err)
		}
		order, err := s.GetOrderByUID(ctx, uid)
		if err != nil {
			log.Printf("пропускаем заказ %s: %v", uid, err)
			continue
		}
		s.mu.Lock()
		s.cache[uid] = order
		s.mu.Unlock()
		count++
	}
	log.Printf("Прогрузили в кэш %d заказов", count)
	return nil
}

// SaveOrder сохраняет все части заказа в рамках одной транзакции.
func (s *Store) SaveOrder(ctx context.Context, order model.OrderData) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Сохраняем информацию о доставке
	_, err = tx.Exec(ctx,
		`INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении доставки: %w", err)
	}

	// 2. Сохраняем информацию об оплате
	_, err = tx.Exec(ctx,
		`INSERT INTO payment (transaction_id, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee,
	)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении оплаты: %w", err)
	}

	// 3. Сохраняем основную информацию о заказе
	_, err = tx.Exec(ctx,
		`INSERT INTO orders (order_uid, track_number, entry, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, locale, internal_signature)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.TrackNumber, order.Entry, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard, order.Locale, order.InternalSignature,
	)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении заказа: %w", err)
	}

	// 4. Сохраняем товары
	for _, item := range order.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
		)
		if err != nil {
			return fmt.Errorf("ошибка при сохранении товара (chrt_id %d): %w", item.ChrtID, err)
		}
	}

	// Если все успешно, коммитим транзакцию
	return tx.Commit(ctx)
}

// GetOrderByUID получает полную информацию о заказе по его UID.
func (s *Store) GetOrderByUID(ctx context.Context, orderUID string) (*model.OrderData, error) {
	// 1. Пытаемся прочитать из кэша (read‑lock)
	s.mu.RLock()
	if order, ok := s.cache[orderUID]; ok {
		s.mu.RUnlock()
		return order, nil
	}
	s.mu.RUnlock()

	// 2. Если в кэше нет — читаем из базы
	order := &model.OrderData{}
	err := s.DB.QueryRow(ctx,
		`SELECT order_uid, track_number, entry, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, locale, internal_signature
		 FROM orders WHERE order_uid = $1`,
		orderUID,
	).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard, &order.Locale, &order.InternalSignature,
	)
	if err != nil {
		return nil, fmt.Errorf("заказ с UID %s не найден: %w", orderUID, err)
	}

	// 3. Получаем информацию о доставке
	err = s.DB.QueryRow(ctx,
		`SELECT name, phone, zip, city, address, region, email FROM delivery WHERE order_uid = $1`,
		orderUID,
	).Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
		&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("не найдена информация о доставке для заказа %s: %w", orderUID, err)
	}

	// 4. Получаем информацию об оплате
	err = s.DB.QueryRow(ctx,
		`SELECT transaction_id, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment WHERE transaction_id = $1`,
		orderUID,
	).Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
		&order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		return nil, fmt.Errorf("не найдена информация об оплате для заказа %s: %w", orderUID, err)
	}

	// 5. Получаем список товаров
	rows, err := s.DB.Query(ctx,
		`SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		 FROM items WHERE order_uid = $1`,
		orderUID,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении товаров для заказа %s: %w", orderUID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.Item
		if err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale,
			&item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании товара заказа: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	// 6. Кладём в кэш (write‑lock)
	s.mu.Lock()
	s.cache[orderUID] = order
	s.mu.Unlock()

	return order, nil
}
