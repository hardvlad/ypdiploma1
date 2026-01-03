// Package handler содержит типы данных и сервисы для обработки запросов
package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"
)

// Handlers структура данных для хранения конфигурации и объектов
type Handlers struct {
	Conf   *config.Config
	Store  repository.StorageInterface
	Logger *zap.SugaredLogger
}

type commonResponse struct {
	isError     bool
	message     string
	redirectURL string
	code        int
}

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

// NewServices создание обработчиков запросов
func NewServices(ctx context.Context, mux *chi.Mux, conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger, ch chan string, wg *sync.WaitGroup, numWorkers int) {
	handlersData := Handlers{
		Conf:   conf,
		Store:  store,
		Logger: sugarLogger,
	}

	CreateWorkers(ctx, numWorkers, handlersData, ch, wg)

	mux.Post(`/api/user/register`, createRegisterHandler(handlersData))
	mux.Post(`/api/user/login`, createLoginHandler(handlersData))
	mux.Post(`/api/user/orders`, createPostOrdersHandler(handlersData, ch))

	mux.Get(`/api/user/orders`, createGetOrdersHandler(handlersData))
	mux.Get(`/api/user/balance`, createGetBalanceHandler(handlersData))

	mux.Post(`/api/user/balance/withdraw`, createWithdrawHandler(handlersData))
	mux.Get(`/api/user/withdrawals`, createGetWithdrawalsHandler(handlersData))

}

// writeResponse функция, выводящая ответ
func writeResponse(w http.ResponseWriter, r *http.Request, resp commonResponse) {
	if resp.isError {
		http.Error(w, resp.message, resp.code)
	} else {
		if resp.redirectURL != "" {
			http.Redirect(w, r, resp.redirectURL, resp.code)
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(resp.code)
			_, err := w.Write([]byte(resp.message))
			if err != nil {
				return
			}
		}
	}
}
