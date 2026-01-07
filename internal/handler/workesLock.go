package handler

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	atomicResumeTime atomic.Int64
)

func initResumeLocker() {
	atomicResumeTime.Store(0)
}

func handle429(resp *http.Response, data Handlers) {
	if resp.StatusCode == http.StatusTooManyRequests {

		var waitDuration time.Duration
		if retryAfterStr := resp.Header.Get("Retry-After"); retryAfterStr != "" {
			if seconds, err := strconv.Atoi(retryAfterStr); err == nil {
				waitDuration = time.Duration(seconds) * time.Second
				data.Logger.Debugw("Status 429", "event", "Получен статус 429, нужно подождать", "seconds", waitDuration)
			} else {
				waitDuration = 5 * time.Second
				data.Logger.Debugw("Status 429", "event", "Получен статус 429, Retry-After не установлен, нужно подождать", "seconds", waitDuration)
			}
		} else {
			waitDuration = 5 * time.Second
		}

		atomicResumeTime.Store(time.Now().Add(waitDuration).UnixNano())
	}
}

func checkAndPause() {
	loadedNano := atomicResumeTime.Load()
	isRateLimited := false
	if loadedNano > 0 {
		isRateLimited = true
	}
	if isRateLimited {
		resumeTime := time.Unix(0, loadedNano)
		waitTime := time.Until(resumeTime)
		if waitTime > 0 {
			<-time.NewTimer(waitTime).C
		}
		atomicResumeTime.Store(0)
	}
}
