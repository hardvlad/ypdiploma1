package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hardvlad/ypdiploma1/internal/retry"
)

func CreateWorkers(ctx context.Context, numWorkers int, data Handlers, ch chan string, wg *sync.WaitGroup) {
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go accrualsWorker(ctx, i, data, ch, wg)
	}
}

// accrualsWorker воркер, слушающий канал, в который поступают номера заказов для обработки
func accrualsWorker(ctx context.Context, id int, data Handlers, ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case orderNumber := <-ch:
			err := processOrderAccruals(ctx, data, orderNumber)
			if err != nil {
				data.Logger.Errorw("accrualsWorker: processOrderAccruals error", "id", id, "orderNumber", orderNumber, "error", err)
			}
		case <-ctx.Done():
			data.Logger.Infow("accrualsWorker: shutting down", "id", id)
			return
		}
	}
}

// processOrderAccruals функция получения статуса заказа и начислений бонусов из внешнего сервиса
func processOrderAccruals(ctx context.Context, data Handlers, number string) error {
	checkAndPause()
	_ = data.Store.SetOrderStatusAccrual(ctx, number, "PROCESSING", 0)

	accrualURL, err := url.JoinPath(data.Conf.AccrualAddress, "/api/orders/", number)
	if err != nil {
		return err
	}

	status, accrual, err := fetchOrderAccruals(data, accrualURL)
	if err != nil {
		return err
	}

	return data.Store.SetOrderStatusAccrual(ctx, number, status, accrual)
}

// fetchOrderAccruals функция, в которой происходит обращение к внешнему сервису начислений
// и ожидающая окончание начислений, периодически запрашивая внешний сервис
func fetchOrderAccruals(data Handlers, url string) (string, float64, error) {
	var status string
	var accrual float64

	for {
		data.Logger.Infow("Getting accruals", "url", url)
		response, err := retry.Retry(3, 2, func() (*http.Response, error) { return http.Get(url) })
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
