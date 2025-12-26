package services

import (
	"io"
	"net/http"

	"github.com/hardvlad/ypdiploma1/internal/util"
)

func createPostOrdersHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: "can't read body",
				code:    http.StatusBadRequest,
			})
			return
		}

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
		}

		orderNumber := string(bodyBytes)
		if !util.CheckNumberLuhn(orderNumber) {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnprocessableEntity),
				code:    http.StatusUnprocessableEntity,
			})
			return
		}

		existingOrderUserID, err := data.Store.GetUserIDOfOrder(orderNumber)
		if err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		if existingOrderUserID != 0 {
			if existingOrderUserID == userID {
				writeResponse(w, r, commonResponse{
					isError: true,
					message: http.StatusText(http.StatusOK),
					code:    http.StatusOK,
				})
			} else {
				writeResponse(w, r, commonResponse{
					isError: true,
					message: http.StatusText(http.StatusConflict),
					code:    http.StatusConflict,
				})
			}
			return
		}

		err = data.Store.InsertNewOrder(orderNumber, userID)
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
			message: http.StatusText(http.StatusAccepted),
			code:    http.StatusAccepted,
		})
	}
}
