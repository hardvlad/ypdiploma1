// Package handler создает хендлер для обработки запросов
package handler

import (
	"net/http"
	"sync"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

// NewHandlers получение основного хендлера для обработки запросов
func NewHandlers(conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger, ch chan string, wg *sync.WaitGroup, numWorkers int) http.Handler {
	mux := chi.NewRouter()
	NewServices(mux, conf, store, sugarLogger, ch, wg, numWorkers)
	return mux
}
