// Package server модуль запуска сервиса
package server

import (
	"net/http"
)

func StartServer(srv *http.Server) error {
	return srv.ListenAndServe()
}
