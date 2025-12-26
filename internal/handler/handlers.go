package handler

import (
	"net/http"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Conf   *config.Config
	Store  repository.StorageInterface
	Logger *zap.SugaredLogger
}

func NewHandlers(conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger) http.Handler {

	mux := chi.NewRouter()

	handlersData := Handlers{
		Conf:   conf,
		Store:  store,
		Logger: sugarLogger,
	}

	mux.Post(`/api/user/register`, createPostHandler(handlersData))
	mux.Post(`/api/user/login`, createPostHandler(handlersData))
	mux.Post(`/api/user/orders`, createPostHandler(handlersData))

	mux.Get(`/api/user/orders`, createGetHandler(handlersData))
	mux.Get(`/api/user/balance`, createGetHandler(handlersData))

	mux.Post(`/api/user/balance/withdraw`, createPostHandler(handlersData))
	mux.Get(`/api/user/withdrawals`, createGetHandler(handlersData))

	return mux
}

func createPostHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func createGetHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}
