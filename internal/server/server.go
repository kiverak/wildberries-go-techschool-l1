package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"l1/internal/model"
)

// OrderGetter определяет интерфейс для получения заказа.
// Это позволяет мокировать хранилище в тестах.
type OrderGetter interface {
	GetOrderByUID(ctx context.Context, orderUID string) (*model.OrderData, error)
}

type Server struct {
	store OrderGetter
}

func New(store OrderGetter) *Server {
	return &Server{store: store}
}

func (s *Server) Start(addr string) error {
	// API
	mux := http.NewServeMux()
	mux.HandleFunc("/order/", s.handleGetOrder)

	// Статика — всё, что лежит в ./web (после сборки фронтенда)
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs) // теперь / и прочие пути пойдут в папку web

	log.Printf("Веб-сервер запущен на http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
		return err
	}

	return nil
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	orderUID := r.URL.Path[len("/order/"):]
	if orderUID == "" {
		http.Error(w, "Не указан UID заказа", http.StatusBadRequest)
		return
	}

	order, err := s.store.GetOrderByUID(context.Background(), orderUID)
	if err != nil {
		log.Printf("Ошибка получения заказа %s: %v", orderUID, err)
		http.Error(w, "Заказ не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(order)
	if err != nil {
		log.Printf("Ошибка кодирования заказа %s: %v", orderUID, err)
		return
	}
}
