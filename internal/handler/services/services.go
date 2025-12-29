// Package services содержит типы данных и сервисы для обработки запросов
package services

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"
)

type contextKey string

// UserIDKey поле в контексте запроса для UserID
const UserIDKey contextKey = "user_id"

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
func NewServices(mux *chi.Mux, conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger) {
	handlersData := Handlers{
		Conf:   conf,
		Store:  store,
		Logger: sugarLogger,
	}

	ch := make(chan string, 100)
	go accrualsWorker(handlersData, ch)

	mux.Post(`/api/user/register`, createRegisterHandler(handlersData))
	mux.Post(`/api/user/login`, createLoginHandler(handlersData))
	mux.Post(`/api/user/orders`, createPostOrdersHandler(handlersData, ch))

	mux.Get(`/api/user/orders`, createGetOrdersHandler(handlersData))
	mux.Get(`/api/user/balance`, createGetBalanceHandler(handlersData))

	mux.Post(`/api/user/balance/withdraw`, createWithdrawHandler(handlersData))
	mux.Get(`/api/user/withdrawals`, createGetWithdrawalsHandler(handlersData))

}

// accrualsWorker воркер, слушающий канал, в который поступают номера заказов для обработки
func accrualsWorker(data Handlers, ch chan string) {
	for orderNumber := range ch {
		err := processOrderAccruals(data, orderNumber)
		if err != nil {
			data.Logger.Errorw("accrualsWorker: processOrderAccruals error", "orderNumber", orderNumber, "error", err)
		}
	}
}

// processOrderAccruals функция получения статуса заказа и начислений бонусов из внешнего сервиса
func processOrderAccruals(data Handlers, number string) error {
	_ = data.Store.SetOrderStatusAccrual(number, "PROCESSING", 0)

	accrualURL, err := url.JoinPath(data.Conf.AccrualAddress, "/api/orders/", number)
	if err != nil {
		return err
	}

	status, accrual, err := fetchOrderAccruals(data, accrualURL)
	if err != nil {
		return err
	}

	return data.Store.SetOrderStatusAccrual(number, status, accrual)
}

// fetchOrderAccruals функция, в которой происходит обращение к внешнему сервису начислений
// и ожидающая окончание начислений, периодически запрашивая внешний сервис
func fetchOrderAccruals(data Handlers, url string) (string, float64, error) {
	var status string
	var accrual float64

	for {
		data.Logger.Infow("Getting accruals", "url", url)
		response, err := http.Get(url)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "fetchOrderAccruals - http.Get error", "url", url)
			return "", 0, err
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusTooManyRequests {
			waitTime := response.Header.Get("Retry-After")
			data.Logger.Debugw("fetchOrderAccruals - received 429 Too Many Requests", "url", url, "waitTime", waitTime)
			waitSeconds, err := strconv.Atoi(waitTime)
			if err != nil {
				waitSeconds = 1
			}

			time.Sleep(time.Second * time.Duration(waitSeconds))
			continue
		}

		if response.StatusCode == http.StatusNoContent {
			status = "NEW"
			accrual = 0
			break
		}

		if response.StatusCode == http.StatusOK {
			var resp AccrualResponse
			dec := json.NewDecoder(response.Body)
			if err := dec.Decode(&resp); err != nil {
				time.Sleep(time.Millisecond * 100)
				continue
			}

			if resp.Status == "INVALID" || resp.Status == "PROCESSED" {
				accrual = resp.Accrual
				status = resp.Status
				break
			}
		}
	}
	return status, accrual, nil
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
