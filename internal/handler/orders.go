// Package handler содержит обработчики для создания заказа для получения начислений
// и получения списка заказов пользователя
package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/hardvlad/ypdiploma1/internal/util"
)

// createPostOrdersHandler создает обработчик для сохранения заказа
// для дальнейшей обработки воркером - получение начислений из сторонней системы
func createPostOrdersHandler(data Handlers, ch chan string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// номер заказа передается в теле - получаем его
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: "can't read body",
				code:    http.StatusBadRequest,
			})
			return
		}

		// получаем userID из контекста
		userID, ok := getUserIDFromRequest(r)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		// проверяем номер заказа по алгоритму Луна - последняя цифра - контрольная сумма
		orderNumber := string(bodyBytes)
		if !util.CheckNumberLuhn(orderNumber) {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnprocessableEntity),
				code:    http.StatusUnprocessableEntity,
			})
			return
		}

		// проверяем есть ли уже заказ с таким номером в базе данных и получаем пользователя, создавшего заказ
		existingOrderUserID, err := data.Store.GetUserIDOfOrder(r.Context(), orderNumber)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// если заказ уже существует - выводим статусы
		if existingOrderUserID != 0 {
			if existingOrderUserID == userID {
				writeResponse(w, r, commonResponse{
					isError: false,
					message: http.StatusText(http.StatusOK),
					code:    http.StatusOK,
				})
			} else {
				writeResponse(w, r, commonResponse{
					isError: false,
					message: http.StatusText(http.StatusConflict),
					code:    http.StatusConflict,
				})
			}
			return
		}

		// сохраняем новый заказ в базе данных
		err = data.Store.InsertNewOrder(r.Context(), orderNumber, userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// отправляем номер заказа в канал для дальнейшей обработки воркером
		ch <- orderNumber

		writeResponse(w, r, commonResponse{
			isError: false,
			message: http.StatusText(http.StatusAccepted),
			code:    http.StatusAccepted,
		})
	}
}

// createGetOrdersHandler создает обработчик для получения списка заказов пользователя
func createGetOrdersHandler(data Handlers) http.HandlerFunc {
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

		// получаем список заказов из базы данных
		orders, err := data.Store.GetOrders(userID)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// если список заказов пустой - выводим StatusNoContent
		if len(orders) == 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusNoContent),
				code:    http.StatusNoContent,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(orders)
	}
}
