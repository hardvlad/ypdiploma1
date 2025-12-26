package repository

import (
	"errors"
)

type StorageInterface interface {
	GetUserIDByLogin(login string) (int, error)
	CreateUser(login string, pwdHash string) (int, error)
	GetUserIDPasswordHashByLogin(login string) (int, string, error)
	GetUserIDOfOrder(orderNumber string) (int, error)
	InsertNewOrder(orderNumber string, userID int) error
}

var ErrorKeyExists = errors.New("key already exists")
