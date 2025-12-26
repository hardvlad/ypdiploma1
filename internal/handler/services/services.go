package services

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"
)

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

func NewServices(mux *chi.Mux, conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger) {
	handlersData := Handlers{
		Conf:   conf,
		Store:  store,
		Logger: sugarLogger,
	}

	mux.Post(`/api/user/register`, createRegisterHandler(handlersData))
	mux.Post(`/api/user/login`, createPostHandler(handlersData))
	mux.Post(`/api/user/orders`, createPostHandler(handlersData))

	mux.Get(`/api/user/orders`, createGetHandler(handlersData))
	mux.Get(`/api/user/balance`, createGetHandler(handlersData))

	mux.Post(`/api/user/balance/withdraw`, createPostHandler(handlersData))
	mux.Get(`/api/user/withdrawals`, createGetHandler(handlersData))

}

func createPostHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func createGetHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

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
