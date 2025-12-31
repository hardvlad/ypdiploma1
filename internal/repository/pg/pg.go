// Package pg реализация интерфейса хранения данных для базы данных Postgres
package pg

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"
)

// Storage тип, содержащий данные, необходимые для работы интерфейса и логирования
type Storage struct {
	DBConn *sql.DB
	mu     sync.RWMutex
	logger *zap.SugaredLogger
}

// NewPGStorage создание объекта хранилища Postgres
func NewPGStorage(dbConn *sql.DB, logger *zap.SugaredLogger) *Storage {
	return &Storage{DBConn: dbConn, logger: logger}
}

// GetUserIDByLogin функция получение ID пользователя по его логину
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

// CreateUser функция создание пользователя по его логину и хешу пароля
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

// GetUserIDPasswordHashByLogin функция получение ID пользователя и хеша пароля по его логину
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

// GetUserIDOfOrder функция получение ID пользователя в заказе
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

// InsertNewOrder функция сохранения в базе данных нового заказа
func (s *Storage) InsertNewOrder(orderNumber string, userID int) error {
	_, err := s.DBConn.ExecContext(context.Background(), "INSERT INTO orders (number, user_id, status_id) VALUES ($1, $2, 1)", orderNumber, userID)
	return err
}

// GetOrders функция получения заказов пользователя
func (s *Storage) GetOrders(userID int) ([]repository.OrdersResult, error) {
	rows, err := s.DBConn.Query("SELECT o.number, os.name, o.accrual, o.uploaded_at FROM orders o JOIN statuses os ON o.status_id = os.id WHERE o.user_id = $1 ORDER BY o.uploaded_at DESC", userID)
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

// GetUserBalance функция получения сумм начислений и списаний пользователя
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

	return accrualsSum, withdrawalsSum, nil
}

// InsertWithdrawal функция сохранения в базе данных списания баланса пользователя
func (s *Storage) InsertWithdrawal(orderNumber string, sum float64, userID int) error {
	sqlStmt := `
    INSERT INTO withdrawals (number, amount, user_id) 
    select $3, $2, id from users
    where id = $1 and  coalesce((select sum(accrual) from orders where user_id=users.id),0) -
                          coalesce((select sum(amount) from withdrawals where user_id=users.id),0) >= $2;
`
	tx, err := s.DBConn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(context.Background(), sqlStmt)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(context.Background(), userID, sum, orderNumber); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetWithdrawals функция получения списка списаний пользователя
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

// SetOrderStatusAccrual функция установления статуса заказа и суммы начислений
func (s *Storage) SetOrderStatusAccrual(orderNumber string, status string, accrual float64) error {
	var statusID int
	err := s.DBConn.QueryRowContext(context.Background(), "SELECT id FROM statuses WHERE name = $1", status).Scan(&statusID)
	if err != nil {
		return err
	}

	_, err = s.DBConn.ExecContext(context.Background(), "UPDATE orders SET status_id = $1, accrual = $2 WHERE number = $3", statusID, accrual, orderNumber)
	return err
}
