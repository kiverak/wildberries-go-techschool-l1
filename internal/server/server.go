package server

import (
	"context"
	"encoding/json"
	"l1/internal/database"
	"log"
	"net/http"
)

type Server struct {
	store *database.Store
}

func New(store *database.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Start(addr string) {
	// API
	mux := http.NewServeMux()
	mux.HandleFunc("/order/", s.handleGetOrder)

	// Статика — всё, что лежит в ./web (после сборки фронтенда)
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs) // теперь / и прочие пути пойдут в папку web

	log.Printf("Веб-сервер запущен на http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
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

	//renderTemplate(w, order)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
