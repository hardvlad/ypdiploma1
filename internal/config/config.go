package config

import "github.com/hardvlad/ypdiploma1/internal/config/db"

type Config struct {
	DBConfig    *db.Config
	CookieName  string
	TokenSecret string
}

func NewConfig(dsn string) *Config {
	return &Config{
		DBConfig:    db.NewConfig(dsn),
		CookieName:  "yp_diploma_one_token",
		TokenSecret: "superSecretKey",
	}
}
