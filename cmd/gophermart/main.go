// Основной исполняемый модуль сервиса
package main

import (
	"errors"
	"log"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/handler"
	"github.com/hardvlad/ypdiploma1/internal/logger"
	"github.com/hardvlad/ypdiploma1/internal/repository/pg"
	"github.com/hardvlad/ypdiploma1/internal/server"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// точка входа в приложение
func main() {
	// инициализация логгера
	myLogger, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}

	defer myLogger.Sync()

	// получение аргументов запуска сервиса
	flags := parseFlags()

	sugarLogger := myLogger.Sugar()
	sugarLogger.Infow("Старт сервера", "addr", flags.RunAddress)

	// создание конфига программы с основными аргументами
	conf := config.NewConfig(flags.Dsn, flags.AccrualAddress)

	// инициализация базы данных
	db, err := conf.DBConfig.InitDB()
	if err != nil {
		sugarLogger.Fatalw(err.Error(), "event", "инициализация базы данных")
	}

	// инициализация хранилища
	store := pg.NewPGStorage(db, sugarLogger)
	defer db.Close()

	// выполнение миграций
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		sugarLogger.Fatalw(err.Error(), "event", "подготовка к миграции")
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		sugarLogger.Fatalw(err.Error(), "event", "создание объекта миграции")
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		sugarLogger.Fatalw(err.Error(), "event", "применение миграции")
	}

	// старт сервера на адресе flags.RunAddress
	err = server.StartServer(flags.RunAddress,
		// с логированием
		logger.WithLogging(
			// с middleware проверки авторизации
			handler.AuthorizationMiddleware(
				// с поддержкой декомпрессии запросов
				handler.RequestDecompressHandle(
					// с поддержкой сжатия ответов
					handler.ResponseCompressHandle(
						// создание обработчика запросов
						handler.NewHandlers(conf, store, sugarLogger),
						sugarLogger,
					),
					sugarLogger,
				),
				sugarLogger, conf.CookieName, conf.TokenSecret, db,
			),
			sugarLogger,
		),
	)

	if err != nil {
		sugarLogger.Fatalw(err.Error(), "event", "start server")
	}
}
