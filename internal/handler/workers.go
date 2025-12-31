package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hardvlad/ypdiploma1/internal/retry"
)

func CreateWorkers(numWorkers int, data Handlers, ch chan string, wg *sync.WaitGroup) {
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go accrualsWorker(i, data, ch, wg)
	}
}

// accrualsWorker воркер, слушающий канал, в который поступают номера заказов для обработки
func accrualsWorker(id int, data Handlers, ch chan string, wg *sync.WaitGroup) {
	for orderNumber := range ch {
		err := processOrderAccruals(data, orderNumber)
		if err != nil {
			data.Logger.Errorw("accrualsWorker: processOrderAccruals error", "id", id, "orderNumber", orderNumber, "error", err)
		}
	}
	wg.Done()
}

// processOrderAccruals функция получения статуса заказа и начислений бонусов из внешнего сервиса
func processOrderAccruals(data Handlers, number string) error {
	CheckAndPause()
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
		response, err := retry.Retry(3, 2*time.Second, func() (*http.Response, error) { return http.Get(url) })
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "fetchOrderAccruals - http.Get error", "url", url)
			return "", 0, err
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusTooManyRequests {
			handle429(response, data)
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
