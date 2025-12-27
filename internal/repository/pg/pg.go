package pg

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/hardvlad/ypdiploma1/internal/repository"
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

func (s *Storage) GetOrders(userID int) ([]repository.OrdersResult, error) {
	rows, err := s.DBConn.QueryContext(context.Background(), "SELECT o.number, os.name, o.accrual, o.uploaded_at FROM orders o JOIN statuses os ON o.status_id = os.id WHERE o.user_id = $1 ORDER BY o.uploaded_at DESC", userID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var orders []repository.OrdersResult

	for rows.Next() {
		var order repository.OrdersResult
		err := rows.Scan(&order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *Storage) GetUserBalance(userID int) (float64, float64, error) {
	row := s.DBConn.QueryRowContext(context.Background(), "SELECT (select sum(amount) from withdrawals where user_id=users.id), (select sum(accrual) from orders where user_id=users.id) FROM users WHERE id = $1", userID)
	var withdrawals sql.NullFloat64
	var accruals sql.NullFloat64

	err := row.Scan(&withdrawals, &accruals)
	if err != nil {
		return 0, 0, err
	}

	withdrawalsSum := 0.0
	accrualsSum := 0.0

	if withdrawals.Valid {
		withdrawalsSum = withdrawals.Float64
	}

	if accruals.Valid {
		accrualsSum = accruals.Float64
	}

	return withdrawalsSum, accrualsSum, nil
}

func (s *Storage) InsertWithdrawal(orderNumber string, sum float64, userID int) error {
	_, err := s.DBConn.ExecContext(context.Background(), "INSERT INTO withdrawals (number, amount, user_id) VALUES ($1, $2, $3)", orderNumber, sum, userID)
	return err
}

func (s *Storage) GetWithdrawals(userID int) ([]repository.WithdrawalsResult, error) {
	rows, err := s.DBConn.QueryContext(context.Background(), "SELECT number, amount, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC", userID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	var withdrawals []repository.WithdrawalsResult

	for rows.Next() {
		var withdrawal repository.WithdrawalsResult
		err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}
