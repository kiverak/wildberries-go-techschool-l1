package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ( // Регулярное выражение для проверки email
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Validate проверяет корректность данных в OrderData.
func (o *OrderData) Validate() error {
	if o.OrderUID == "" {
		return errors.New("OrderUID не может быть пустым")
	}
	if o.TrackNumber == "" {
		return errors.New("TrackNumber не может быть пустым")
	}
	if o.Entry == "" {
		return errors.New("Entry не может быть пустым")
	}
	if o.Locale == "" {
		return errors.New("Locale не может быть пустым")
	}
	if o.CustomerID == "" {
		return errors.New("CustomerID не может быть пустым")
	}
	if o.DeliveryService == "" {
		return errors.New("DeliveryService не может быть пустым")
	}
	if o.Shardkey == "" {
		return errors.New("Shardkey не может быть пустым")
	}
	if o.SmID <= 0 {
		return errors.New("SmID должен быть положительным числом")
	}
	if o.DateCreated.IsZero() {
		return errors.New("DateCreated не может быть нулевым")
	}
	if o.OofShard == "" {
		return errors.New("OofShard не может быть пустым")
	}

	if err := o.Delivery.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации Delivery: %w", err)
	}

	if err := o.Payment.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации Payment: %w", err)
	}

	if len(o.Items) == 0 {
		return errors.New("список товаров (Items) не может быть пустым")
	}
	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("ошибка валидации Item #%d: %w", i, err)
		}
	}

	return nil
}

// Validate проверяет корректность данных в Delivery.
func (d *Delivery) Validate() error {
	if d.Name == "" {
		return errors.New("Delivery.Name не может быть пустым")
	}
	if d.Phone == "" {
		return errors.New("Delivery.Phone не может быть пустым")
	}
	if d.Zip == "" {
		return errors.New("Delivery.Zip не может быть пустым")
	}
	if d.City == "" {
		return errors.New("Delivery.City не может быть пустым")
	}
	if d.Address == "" {
		return errors.New("Delivery.Address не может быть пустым")
	}
	if d.Region == "" {
		return errors.New("Delivery.Region не может быть пустым")
	}
	if d.Email == "" {
		return errors.New("Delivery.Email не может быть пустым")
	}
	if !emailRegex.MatchString(d.Email) {
		return errors.New("Delivery.Email имеет неверный формат")
	}
	return nil
}

// Validate проверяет корректность данных в Payment.
func (p *Payment) Validate() error {
	if p.Transaction == "" {
		return errors.New("Payment.Transaction не может быть пустым")
	}
	if p.Currency == "" {
		return errors.New("Payment.Currency не может быть пустым")
	}
	if !isValidCurrency(p.Currency) {
		return errors.New("Payment.Currency имеет неверное значение")
	}

	if p.Provider == "" {
		return errors.New("Payment.Provider не может быть пустым")
	}
	if p.Amount < 0 {
		return errors.New("Payment.Amount не может быть отрицательным")
	}
	if p.PaymentDt < 0 {
		return errors.New("Payment.PaymentDt не может быть отрицательным")
	}
	if p.Bank == "" {
		return errors.New("Payment.Bank не может быть пустым")
	}
	if p.DeliveryCost < 0 {
		return errors.New("Payment.DeliveryCost не может быть отрицательным")
	}
	if p.GoodsTotal < 0 {
		return errors.New("Payment.GoodsTotal не может быть отрицательным")
	}
	if p.CustomFee < 0 {
		return errors.New("Payment.CustomFee не может быть отрицательным")
	}
	return nil
}

// isValidCurrency - Валидация валюты по списку допустимых значений
func isValidCurrency(currency string) bool {
	switch strings.ToUpper(currency) {
	case "USD", "RUB", "EUR": // Пример допустимых валют
		return true
	default:
		return false
	}
}

// Validate проверяет корректность данных в Item.
func (i *Item) Validate() error {
	if i.ChrtID <= 0 {
		return errors.New("Item.ChrtID должен быть положительным числом")
	}
	if i.TrackNumber == "" {
		return errors.New("Item.TrackNumber не может быть пустым")
	}
	if i.Price < 0 {
		return errors.New("Item.Price не может быть отрицательным")
	}
	if i.Rid == "" {
		return errors.New("Item.Rid не может быть пустым")
	}
	if i.Name == "" {
		return errors.New("Item.Name не может быть пустым")
	}
	if i.Sale < 0 {
		return errors.New("Item.Sale не может быть отрицательным")
	}
	if i.Size == "" {
		return errors.New("Item.Size не может быть пустым")
	}
	if i.TotalPrice < 0 {
		return errors.New("Item.TotalPrice не может быть отрицательным")
	}
	if i.NmID <= 0 {
		return errors.New("Item.NmID должен быть положительным числом")
	}
	if i.Brand == "" {
		return errors.New("Item.Brand не может быть пустым")
	}
	if i.Status < 0 {
		return errors.New("Item.Status не может быть отрицательным")
	}
	return nil
}
