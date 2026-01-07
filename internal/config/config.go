// Package config создание объекта конфигурации сервиса
package config

import "github.com/hardvlad/ypdiploma1/internal/config/db"

// Config тип описывающий структуру конфига приложения
type Config struct {
	DBConfig       *db.Config
	CookieName     string
	TokenSecret    string
	AccrualAddress string
}

// NewConfig создание и наполнение структуры конфига приложения
func NewConfig(dsn string, accrualAddress string) *Config {
	return &Config{
		DBConfig:       db.NewConfig(dsn),
		CookieName:     "yp_diploma_one_token",
		TokenSecret:    "superSecretKey",
		AccrualAddress: accrualAddress,
	}
}
