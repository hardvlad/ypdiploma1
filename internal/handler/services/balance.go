package services

import (
	"encoding/json"
	"net/http"
)

type GetBalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func createGetBalanceHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		var balance GetBalanceResponse
		var err error
		balance.Current, balance.Withdrawn, err = data.Store.GetUserBalance(userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(balance)
	}
}
