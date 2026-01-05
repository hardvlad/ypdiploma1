// Package repository описание интерфейса хранения данных и типов для работы с базой данных
package repository

import (
	"context"
	"time"
)

// OrdersResult тип, описывающий результат запроса заказов пользователя
type OrdersResult struct {
	OrderNumber string    `json:"number"`
	Status      string    `json:"status"`
	Accrual     float64   `json:"accrual,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// WithdrawalsResult тип, описывающий результат запроса списания бонусов пользователя
type WithdrawalsResult struct {
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type StorageInterface interface {
	// GetUserIDByLogin функция получение ID пользователя по его логину
	GetUserIDByLogin(ctx context.Context, login string) (int, error)
	// CreateUser функция создание пользователя по его логину и хешу пароля
	CreateUser(ctx context.Context, login string, pwdHash string) (int, error)
	// GetUserIDPasswordHashByLogin функция получение ID пользователя и хеша пароля по его логину
	GetUserIDPasswordHashByLogin(ctx context.Context, login string) (int, string, error)
	// GetUserIDOfOrder функция получение ID пользователя в заказе
	GetUserIDOfOrder(ctx context.Context, orderNumber string) (int, error)
	// InsertNewOrder функция сохранения в базе данных нового заказа
	InsertNewOrder(ctx context.Context, orderNumber string, userID int) error
	// GetOrders функция получения заказов пользователя
	GetOrders(userID int) ([]OrdersResult, error)
	// GetUserBalance функция получения сумм начислений и списаний пользователя
	GetUserBalance(ctx context.Context, userID int) (float64, float64, error)
	// InsertWithdrawal функция сохранения в базе данных списания баланса пользователя
	InsertWithdrawal(ctx context.Context, orderNumber string, sum float64, userID int) error
	// GetWithdrawals функция получения списка списаний пользователя
	GetWithdrawals(ctx context.Context, userID int) ([]WithdrawalsResult, error)
	// SetOrderStatusAccrual функция установления статуса заказа и суммы начислений
	SetOrderStatusAccrual(ctx context.Context, orderNumber string, status string, accrual float64) error
}
