package main

import (
	"errors"
	"log"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/handler"
	"github.com/hardvlad/ypdiploma1/internal/logger"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"github.com/hardvlad/ypdiploma1/internal/repository/pg"
	"github.com/hardvlad/ypdiploma1/internal/server"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	myLogger, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}

	defer myLogger.Sync()

	flags := parseFlags()

	sugarLogger := myLogger.Sugar()
	sugarLogger.Infow("Старт сервера", "addr", flags.RunAddress)

	conf := config.NewConfig(flags.Dsn)

	var store repository.StorageInterface

	db, err := conf.DBConfig.InitDB()

	store = pg.NewPGStorage(db, sugarLogger)
	defer db.Close()

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

	err = server.StartServer(flags.RunAddress,
		logger.WithLogging(
			handler.AuthorizationMiddleware(
				handler.RequestDecompressHandle(
					handler.ResponseCompressHandle(
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
