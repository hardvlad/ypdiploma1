// Основной исполняемый модуль сервиса
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	// получение аргументов запуска сервиса
	flags := parseFlags()

	// инициализация логгера
	myLogger, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	sugarLogger := myLogger.Sugar()
	sugarLogger.Infow("Старт сервера", "addr", flags.RunAddress)
	defer myLogger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	idleConnsClosed := make(chan struct{})

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

	var wg sync.WaitGroup
	numWorkers := 3
	ch := make(chan string, numWorkers)

	mux := logger.WithLogging(
		// с middleware проверки авторизации
		handler.AuthorizationMiddleware(
			// с поддержкой декомпрессии запросов
			handler.RequestDecompressHandle(
				// с поддержкой сжатия ответов
				handler.ResponseCompressHandle(
					// создание обработчика запросов
					handler.NewHandlers(ctx, conf, store, sugarLogger, ch, &wg, numWorkers),
					sugarLogger,
				),
				sugarLogger,
			),
			sugarLogger, conf.CookieName, conf.TokenSecret, db,
		),
		sugarLogger,
	)

	addr := flags.RunAddress
	if addr == "" {
		addr = ":8080"
	}

	// создание HTTP сервера
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		sugarLogger.Infow("Завершение работы сервиса")
		if err := srv.Shutdown(context.Background()); err != nil {
			sugarLogger.Debugw(err.Error(), "event", "shutdown server")
		}

		cancel()
		sugarLogger.Infow("Завершение работы сервиса 2")
		close(idleConnsClosed)
	}()

	// старт сервера на адресе flags.RunAddress
	err = server.StartServer(srv)

	<-idleConnsClosed

	if err != nil {
		sugarLogger.Infow(err.Error(), "event", "start server")
	}

	sugarLogger.Infow("HTTP сервер остановлен")

	close(ch)
	wg.Wait()
}
