package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (s *Storage) Get(key string) (string, bool, bool) {
	row := s.DBConn.QueryRowContext(context.Background(), "SELECT url, is_deleted from saved_links where code = $1 limit 1", key)

	var savedURL string
	var isDeleted bool
	err := row.Scan(&savedURL, &isDeleted)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Debugw(err.Error(), "event", "get from DB", "code", key)
		}
		return "", false, false
	}
	return savedURL, isDeleted, true
}

func (s *Storage) GetCode(url string) (string, bool) {
	row := s.DBConn.QueryRowContext(context.Background(), "SELECT code from saved_links where url = $1 and is_deleted=false limit 1", url)

	var savedCode string
	err := row.Scan(&savedCode)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Debugw(err.Error(), "event", "get code from DB", "url", url)
		}
		return "", false
	}
	return savedCode, true
}

func (s *Storage) Set(key, value string, userID int) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, isDeleted, exists := s.Get(key); exists && !isDeleted {
		return "", false, fmt.Errorf("%w: %s", repository.ErrorKeyExists, key)
	}

	if code, exists := s.GetCode(value); exists {
		return code, true, nil
	}

	_, err := s.DBConn.ExecContext(context.Background(), "INSERT INTO saved_links (code, url, user_id) VALUES ($1, $2, $3)", key, value, userID)
	if err != nil {
		return "", false, err
	}
	return key, false, nil
}

func (s *Storage) GetUserData(userID int) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.DBConn.QueryContext(context.Background(), "SELECT code, url from saved_links where user_id = $1 and is_deleted=false", userID)
	if err != nil {
		s.logger.Debugw(err.Error(), "event", "получение данных пользователя", "user_id", userID)
		return nil, err
	}

	if rows.Err() != nil {
		s.logger.Debugw(rows.Err().Error(), "event", "получение данных пользователя", "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	userData := make(map[string]string)
	for rows.Next() {
		var code, url string
		if err := rows.Scan(&code, &url); err != nil {
			return nil, err
		}
		userData[code] = url
	}
	return userData, nil
}

func (s *Storage) DeleteURLs(codes []string, userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.DBConn.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(context.Background(), "UPDATE saved_links set is_deleted=true WHERE code = $1 AND user_id = $2")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, code := range codes {
		if _, err := stmt.ExecContext(context.Background(), code, userID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
