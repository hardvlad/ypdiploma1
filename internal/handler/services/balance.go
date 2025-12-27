package services

import (
	"encoding/json"
	"net/http"

	"github.com/hardvlad/ypdiploma1/internal/util"
)

type GetBalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
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

		balance.Current -= balance.Withdrawn

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(balance)
	}
}

func createWithdrawHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		var requestData WithdrawRequest

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&requestData); err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		if requestData.OrderNumber == "" || requestData.Sum <= 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		if !util.CheckNumberLuhn(requestData.OrderNumber) {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnprocessableEntity),
				code:    http.StatusUnprocessableEntity,
			})
			return
		}

		accrued, withdrawn, err := data.Store.GetUserBalance(userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		balance := accrued - withdrawn

		if balance < requestData.Sum {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusPaymentRequired),
				code:    http.StatusPaymentRequired,
			})
			return
		}

		err = data.Store.InsertWithdrawal(requestData.OrderNumber, requestData.Sum, userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		writeResponse(w, r, commonResponse{
			isError: false,
			message: http.StatusText(http.StatusOK),
			code:    http.StatusOK,
		})
	}
}

func createGetWithdrawalsHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		withdrawals, err := data.Store.GetWithdrawals(userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		if len(withdrawals) == 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusNoContent),
				code:    http.StatusNoContent,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(withdrawals)
	}
}
