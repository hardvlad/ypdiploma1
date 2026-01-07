// Package handler содержит методы для получения баланса пользователя,
// списка списаний и создание списания
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hardvlad/ypdiploma1/internal/util"
)

// GetBalanceResponse структура, описывающая формат ответа на запрос баланса
type GetBalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// WithdrawRequest структура, описывающая формат запроса на списание
type WithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
}

// createGetBalanceHandler - создание обработчика метода получения баланса
func createGetBalanceHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// получаем userID из контекста
		userID, ok := getUserIDFromRequest(r)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		// получаем баланс пользователя из базы
		var balance GetBalanceResponse
		var err error
		balance.Current, balance.Withdrawn, err = data.Store.GetUserBalance(r.Context(), userID)
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

// createWithdrawHandler - создание обработчика метода списания
func createWithdrawHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// получаем userID из контекста
		userID, ok := r.Context().Value(userIDKey).(int)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		var requestData WithdrawRequest

		// попытка разобрать запрос в структуру
		dec := json.NewDecoder(r.Body)
		// если попытка неудачна - выводим StatusBadRequest и прекращаем обработку
		if err := dec.Decode(&requestData); err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		// проверяем ограничения на данные в запросе
		if requestData.OrderNumber == "" || requestData.Sum <= 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		// проверяем номер заказа по алгоритму Луна - последняя цифра - контрольная сумма
		if !util.CheckNumberLuhn(requestData.OrderNumber) {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnprocessableEntity),
				code:    http.StatusUnprocessableEntity,
			})
			return
		}

		// получаем баланс из базы данных
		accrued, withdrawn, err := data.Store.GetUserBalance(r.Context(), userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		balance := accrued - withdrawn

		// если баланс меньше суммы списания - выводим ошибку
		if balance < requestData.Sum {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusPaymentRequired),
				code:    http.StatusPaymentRequired,
			})
			return
		}

		// сохраняем списание в базе данных
		err = data.Store.InsertWithdrawal(r.Context(), requestData.OrderNumber, requestData.Sum, userID)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "insert withdrawal", "userID", userID, "number", requestData.OrderNumber, "sum", requestData.Sum)
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

// createGetWithdrawalsHandler - создание обработчика метода для получения списка списаний
func createGetWithdrawalsHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// получаем userID из контекста
		userID, ok := getUserIDFromRequest(r)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		// получаем список списаний из базы данных
		withdrawals, err := data.Store.GetWithdrawals(r.Context(), userID)
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
