package server

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"time"

	"l1/internal/database"
	"l1/internal/model"
)

type Server struct {
	store *database.Store
}

func New(store *database.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/order/", s.handleGetOrder)

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

	renderTemplate(w, order)
}

func renderTemplate(w http.ResponseWriter, order *model.OrderData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Добавляем функцию для форматирования Unix-времени
	funcMap := template.FuncMap{
		"unixToTime": func(ts int64) string {
			return time.Unix(ts, 0).Format("02.01.2006 15:04:05")
		},
	}

	tmpl, err := template.New("order.html").
		Funcs(funcMap).
		ParseFiles("internal/server/templates/order.html")
	if err != nil {
		http.Error(w, "Ошибка рендеринга шаблона", http.StatusInternalServerError)
		log.Printf("Ошибка парсинга шаблона: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, order); err != nil {
		log.Printf("Ошибка выполнения шаблона: %v", err)
	}
}
