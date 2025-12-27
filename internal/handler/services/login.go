package services

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hardvlad/ypdiploma1/internal/auth"
	"github.com/hardvlad/ypdiploma1/internal/util"
)

type loginUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func createLoginHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var user loginUser

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&user); err != nil {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

		if user.Password == "" || user.Login == "" {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusBadRequest),
				code:    http.StatusBadRequest,
			})
			return
		}

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

		if userID == 0 {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnauthorized),
				code:    http.StatusUnauthorized,
			})
			return
		}

		ok := util.CheckPasswordHash(user.Password, pwdHash)
		if !ok {
			writeResponse(w, r, commonResponse{
				isError: true,
				message: http.StatusText(http.StatusUnauthorized),
				code:    http.StatusUnauthorized,
			})
			return
		}

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
