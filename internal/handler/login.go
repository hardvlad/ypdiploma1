// Package handler содержит обработчик входа пользователя по имени и паролю
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hardvlad/ypdiploma1/internal/auth"
	"github.com/hardvlad/ypdiploma1/internal/util"
)

// loginUser структура, описывающая формат запроса в JSON
type loginUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// createLoginHandler обработчик входа пользователя по имени и паролю
func createLoginHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var user loginUser

		// попытка разобрать запрос в структуру
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&user); err != nil {
			// если попытка неудачна - выводим StatusBadRequest и прекращаем обработку
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		// проверка на пустые логин и пароль - если пустые, то выводим StatusBadRequest и прекращаем обработку
		if user.Password == "" || user.Login == "" {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		// получение из базы userID и хэша пароля по логину
		userID, pwdHash, err := data.Store.GetUserIDPasswordHashByLogin(user.Login)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "login - get userID error", "login", user.Login)
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// если userID == 0, значит пользователь с таким логином не найден, выводим StatusUnauthorized
		if userID == 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnauthorized),
				code:    http.StatusUnauthorized,
			})
			return
		}

		// проверка пароля, если не совпадает - выводим StatusUnauthorized
		ok := util.CheckPasswordHash(user.Password, pwdHash)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnauthorized),
				code:    http.StatusUnauthorized,
			})
			return
		}

		// создание токена
		token, err := auth.CreateToken(time.Hour*24, userID, data.Conf.TokenSecret)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "login - create token", "login", user.Login, "userID", userID)
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  data.Conf.CookieName,
			Value: token,
		})

		writeResponse(w, r, commonResponse{
			isError: false,
			message: http.StatusText(http.StatusOK),
			code:    http.StatusOK,
		})
	}
}
