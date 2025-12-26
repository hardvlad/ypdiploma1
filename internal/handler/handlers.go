package handler

import (
	"net/http"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/handler/services"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func NewHandlers(conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger) http.Handler {
	mux := chi.NewRouter()
	services.NewServices(mux, conf, store, sugarLogger)
	return mux
}
