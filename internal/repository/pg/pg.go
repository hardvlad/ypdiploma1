package pg

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"go.uber.org/zap"
)

type Storage struct {
	DBConn *sql.DB
	mu     sync.RWMutex
	logger *zap.SugaredLogger
}

func NewPGStorage(dbConn *sql.DB, logger *zap.SugaredLogger) *Storage {
	return &Storage{DBConn: dbConn, logger: logger}
}

func (s *Storage) GetUserIDByLogin(login string) (int, error) {
	row := s.DBConn.QueryRowContext(context.Background(), "SELECT id from users where login = $1", login)

	userID := 0
	err := row.Scan(&userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		return 0, nil
	}
	return userID, nil
}

func (s *Storage) CreateUser(login string, pwdHash string) (int, error) {
	var userID int
	err := s.DBConn.QueryRowContext(
		context.Background(),
		"INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id",
		login,
		pwdHash,
	).Scan(&userID)

	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (s *Storage) GetUserIDPasswordHashByLogin(login string) (int, string, error) {
	row := s.DBConn.QueryRowContext(context.Background(), "SELECT id, password_hash from users where login = $1", login)

	userID := 0
	var pwdHash string
	err := row.Scan(&userID, &pwdHash)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, "", err
		}
		return 0, "", nil
	}
	return userID, pwdHash, nil
}

func (s *Storage) GetUserIDOfOrder(orderNumber string) (int, error) {
	row := s.DBConn.QueryRowContext(context.Background(), "SELECT user_id from orders where number = $1", orderNumber)

	userID := 0
	err := row.Scan(&userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		return 0, nil
	}
	return userID, nil
}

func (s *Storage) InsertNewOrder(orderNumber string, userID int) error {
	_, err := s.DBConn.ExecContext(context.Background(), "INSERT INTO orders (number, user_id, status_id) VALUES ($1, $2, 1)", orderNumber, userID)
	return err
}
