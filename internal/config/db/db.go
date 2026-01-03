// Package db конфигурация для базы данных и подключение к ней
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config тип, описывающий структуру конфига базы данных
type Config struct {
	Dsn string
}

// NewConfig создание и наполнение структуры конфига базы данных
func NewConfig(dsn string) *Config {
	return &Config{
		Dsn: dsn,
	}
}

// InitDB подключение к базе данных, используя конфиг,
// возвращает или подключение к базе данных или ошибку
func (c *Config) InitDB() (*sql.DB, error) {

	if c.Dsn == "" {
		return nil, fmt.Errorf("dsn не задано")
	}

	db, err := sql.Open("pgx", c.Dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка соединения с базой данных: %w", err)
	}

	// Verify the connection is alive
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка пинга базы данных: %w", err)
	}

	return db, nil
}
