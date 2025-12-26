package repository

import (
	"errors"
)

type StorageInterface interface {
	GetUserIDByLogin(login string) (int, error)
	CreateUser(login string, pwdHash string) (int, error)
}

var ErrorKeyExists = errors.New("key already exists")
