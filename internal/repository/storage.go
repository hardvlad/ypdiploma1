package repository

import (
	"errors"
)

type StorageInterface interface {
}

var ErrorKeyExists = errors.New("key already exists")
