package handler

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	isRateLimitedMux sync.Mutex
	isRateLimited    bool
	resumeTime       time.Time
)

func handle429(resp *http.Response, data Handlers) time.Duration {
	if resp.StatusCode == http.StatusTooManyRequests {
		isRateLimitedMux.Lock()
		defer isRateLimitedMux.Unlock()

		var waitDuration time.Duration
		if retryAfterStr := resp.Header.Get("Retry-After"); retryAfterStr != "" {
			if seconds, err := strconv.Atoi(retryAfterStr); err == nil {
				waitDuration = time.Duration(seconds) * time.Second
			} else {
				waitDuration = 5 * time.Second
			}
		} else {
			waitDuration = 5 * time.Second
		}

		resumeTime = time.Now().Add(waitDuration)
		isRateLimited = true
		data.Logger.Debugw("Status 429", "event", "Получен статус 429, засыпаем на", "seconds", waitDuration)
		return waitDuration
	}
	return 0
}

func CheckAndPause() {
	isRateLimitedMux.Lock()
	if isRateLimited {
		waitTime := time.Until(resumeTime)
		if waitTime > 0 {
			isRateLimitedMux.Unlock()
			select {
			case <-time.After(waitTime):
				isRateLimitedMux.Lock()
				isRateLimited = false
				isRateLimitedMux.Unlock()
				return
			}
		}
	}
	isRateLimitedMux.Unlock()
}
