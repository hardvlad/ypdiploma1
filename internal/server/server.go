package server

import (
	"net/http"
)

func StartServer(addr string, mux http.Handler) error {
	if addr == "" {
		addr = ":8080"
	}
	return http.ListenAndServe(addr, mux)
}
