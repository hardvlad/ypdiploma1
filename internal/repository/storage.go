package repository

import "time"

type OrdersResult struct {
	OrderNumber string    `json:"number"`
	Status      string    `json:"status"`
	Accrual     float64   `json:"accrual,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type StorageInterface interface {
	GetUserIDByLogin(login string) (int, error)
	CreateUser(login string, pwdHash string) (int, error)
	GetUserIDPasswordHashByLogin(login string) (int, string, error)
	GetUserIDOfOrder(orderNumber string) (int, error)
	InsertNewOrder(orderNumber string, userID int) error
	GetOrders(userID int) ([]OrdersResult, error)
}
