// Package handler содержит реализацию метода регистрации пользователя
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hardvlad/ypdiploma1/internal/auth"
	"github.com/hardvlad/ypdiploma1/internal/util"
)

// registerUser структура, описывающая формат запроса в JSON
type registerUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// createRegisterHandler обработчик регистрации пользователя по имени и паролю
func createRegisterHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var user registerUser

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

		// получение из базы userID по логину
		userID, err := data.Store.GetUserIDByLogin(user.Login)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "register - get userID error", "login", user.Login)
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// если пользователь найден - выдаем ошибку
		if userID > 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusConflict),
				code:    http.StatusConflict,
			})
			return
		}

		// создание хэша пароля
		pwdHash, err := util.HashPassword(user.Password)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "register - get password hash", "login", user.Login)
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// сохранение пользователя в базе данных
		userID, err = data.Store.CreateUser(user.Login, pwdHash)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "register - create user", "login", user.Login)
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		// создание токена
		token, err := auth.CreateToken(time.Hour*24, userID, data.Conf.TokenSecret)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "register - create token", "login", user.Login, "userID", userID)
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
