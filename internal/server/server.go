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
	// Добавляем функцию для форматирования Unix-времени
	funcMap := template.FuncMap{
		"unixToTime": func(ts int64) string {
			return time.Unix(ts, 0).Format("02.01.2006 15:04:05")
		},
	}

	tmpl, err := template.New("order").Funcs(funcMap).Parse(orderTemplate)
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

const orderTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Заказ {{.OrderUID}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; background-color: #f8f9fa; color: #212529; }
        .container { max-width: 960px; margin: 20px auto; background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1, h2 { color: #343a40; }
        .grid-container { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 20px; }
        .card { padding: 15px; border: 1px solid #dee2e6; border-radius: 4px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { border: 1px solid #dee2e6; padding: 12px; text-align: left; }
        th { background-color: #e9ecef; }
        p { margin: 5px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Заказ: {{.OrderUID}}</h1>
        <p><strong>Track Number:</strong> {{.TrackNumber}}</p>
        <p><strong>Customer ID:</strong> {{.CustomerID}}</p>
        <p><strong>Дата создания:</strong> {{.DateCreated.Format "02.01.2006 15:04:05 MST"}}</p>

        <div class="grid-container">
            <div class="card">
                <h2>Информация о доставке</h2>
                <p><strong>Имя:</strong> {{.Delivery.Name}}</p>
                <p><strong>Телефон:</strong> {{.Delivery.Phone}}</p>
                <p><strong>Email:</strong> {{.Delivery.Email}}</p>
                <p><strong>Город:</strong> {{.Delivery.City}}, {{.Delivery.Region}}</p>
                <p><strong>Адрес:</strong> {{.Delivery.Address}}, {{.Delivery.Zip}}</p>
            </div>
            <div class="card">
                <h2>Информация об оплате</h2>
                <p><strong>Транзакция:</strong> {{.Payment.Transaction}}</p>
                <p><strong>Валюта:</strong> {{.Payment.Currency}}</p>
                <p><strong>Сумма:</strong> {{.Payment.Amount}}</p>
                <p><strong>Стоимость доставки:</strong> {{.Payment.DeliveryCost}}</p>
                <p><strong>Всего за товары:</strong> {{.Payment.GoodsTotal}}</p>
                <p><strong>Дата оплаты:</strong> {{.Payment.PaymentDt | unixToTime}}</p>
                <p><strong>Банк:</strong> {{.Payment.Bank}}</p>
                <p><strong>Провайдер:</strong> {{.Payment.Provider}}</p>
            </div>
        </div>

        <h2>Товары в заказе</h2>
        <table>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Название</th>
                    <th>Бренд</th>
                    <th>Цена</th>
                    <th>Скидка (%)</th>
                    <th>Итоговая цена</th>
                </tr>
            </thead>
            <tbody>
                {{range .Items}}
                <tr>
                    <td>{{.ChrtID}}</td>
                    <td>{{.Name}}</td>
                    <td>{{.Brand}}</td>
                    <td>{{.Price}}</td>
                    <td>{{.Sale}}</td>
                    <td>{{.TotalPrice}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`
