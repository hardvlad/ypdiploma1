package main

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/handler"
	"github.com/hardvlad/ypdiploma1/internal/logger"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"github.com/hardvlad/ypdiploma1/internal/repository/pg"
	"github.com/hardvlad/ypdiploma1/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type want struct {
	code        int
	response    string
	contentType string
}

type test struct {
	name   string
	method string
	target string
	body   string
	want   want
}

var (
	globalFlags  programFlags
	globalDB     *sql.DB
	globalMux    http.Handler
	globalLogger *zap.SugaredLogger
	globalZap    *zap.Logger
)

func prepareMux() (*sql.DB, http.Handler, error) {
	myLogger, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}

	globalZap = myLogger

	flags := parseFlags()

	sugarLogger := myLogger.Sugar()
	sugarLogger.Infow("Старт сервера", "addr", flags.RunAddress)

	conf := config.NewConfig(flags.Dsn, flags.AccrualAddress)

	var store repository.StorageInterface

	db, err := conf.DBConfig.InitDB()
	if err != nil {
		sugarLogger.Fatalw(err.Error(), "event", "инициализация базы данных")
		return nil, nil, err
	}

	store = pg.NewPGStorage(db, sugarLogger)
	mux := logger.WithLogging(
		handler.AuthorizationMiddleware(
			handler.RequestDecompressHandle(
				handler.ResponseCompressHandle(
					handler.NewHandlers(conf, store, sugarLogger),
					sugarLogger,
				),
				sugarLogger,
			),
			sugarLogger, conf.CookieName, conf.TokenSecret, db,
		),
		sugarLogger,
	)

	globalFlags = flags
	globalDB = db
	globalMux = mux
	globalLogger = sugarLogger

	return db, mux, nil
}

func TestRegister(t *testing.T) {
	login := "testuser" + util.GenerateRandomString(6)
	tests := []test{
		{
			name:   "register empty login #1",
			method: http.MethodPost,
			target: "/api/user/register",
			body:   `{"login":"","password":"password"}`,
			want: want{
				code: 400,
			},
		},
		{
			name:   "register empty password #1",
			method: http.MethodPost,
			target: "/api/user/register",
			body:   `{"login":"login","password":""}`,
			want: want{
				code: 400,
			},
		},
		{
			name:   "register new user #1",
			method: http.MethodPost,
			target: "/api/user/register",
			body:   `{"login":"` + login + `","password":"xxxxyyyy"}`,
			want: want{
				code: 200,
			},
		},
		{
			name:   "register user conflict #1",
			method: http.MethodPost,
			target: "/api/user/register",
			body:   `{"login":"` + login + `","password":"xxxxyyyy"}`,
			want: want{
				code: 409,
			},
		},
	}

	_, mux, err := prepareMux()
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.target, strings.NewReader(test.body))

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, request)

			res := w.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			res.Body.Close()
		})
	}
}

func TestRegisterLogin(t *testing.T) {
	badLogin := "testuser" + util.GenerateRandomString(10)
	login := "testuser" + util.GenerateRandomString(6)
	tests := []test{
		{
			name:   "login - register new user #1",
			method: http.MethodPost,
			target: "/api/user/register",
			body:   `{"login":"` + login + `","password":"xxxxyyyy"}`,
			want: want{
				code: 200,
			},
		},
		{
			name:   "login not existing user #1",
			method: http.MethodPost,
			target: "/api/user/login",
			body:   `{"login":"` + badLogin + `","password":"xxxxyyyy"}`,
			want: want{
				code: 401,
			},
		},
		{
			name:   "login correct #1",
			method: http.MethodPost,
			target: "/api/user/login",
			body:   `{"login":"` + login + `","password":"xxxxyyyy"}`,
			want: want{
				code: 200,
			},
		},
	}

	mux := globalMux

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.target, strings.NewReader(test.body))

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, request)

			res := w.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			res.Body.Close()
		})
	}
}

func TestOrders(t *testing.T) {
	login := "testuser" + util.GenerateRandomString(6)

	orderNumber := util.DigitString(6, 7)
	number, err := strconv.Atoi(orderNumber)
	require.NoError(t, err)
	checkNumber := util.CalcChecksumLuhn(number)
	badOrderNumber := orderNumber
	if checkNumber != 0 {
		badOrderNumber = badOrderNumber + strconv.Itoa(checkNumber)
		checkNumber = 10 - checkNumber
	} else {
		badOrderNumber = badOrderNumber + "9"
	}
	orderNumber = orderNumber + strconv.Itoa(checkNumber)

	tests := []test{
		{
			name:   "orders - register new user #1",
			method: http.MethodPost,
			target: "/api/user/register",
			body:   `{"login":"` + login + `","password":"xxxxyyyy"}`,
			want: want{
				code: 200,
			},
		},
		{
			name:   "order post no cookie #1",
			method: http.MethodPost,
			target: "/api/user/orders",
			body:   orderNumber,
			want: want{
				code: http.StatusUnauthorized,
			},
		},
		{
			name:   "order get balance no cookie #1",
			method: http.MethodGet,
			target: "/api/user/balance",
			body:   "",
			want: want{
				code: http.StatusUnauthorized,
			},
		},
		{
			name:   "order get no orders #1",
			method: http.MethodGet,
			target: "/api/user/orders",
			body:   "",
			want: want{
				code: http.StatusNoContent,
			},
		},
		{
			name:   "order post #1",
			method: http.MethodPost,
			target: "/api/user/orders",
			body:   orderNumber,
			want: want{
				code: http.StatusAccepted,
			},
		},
		{
			name:   "order post bad number #1",
			method: http.MethodPost,
			target: "/api/user/orders",
			body:   badOrderNumber,
			want: want{
				code: http.StatusUnprocessableEntity,
			},
		},
		{
			name:   "order get orders #1",
			method: http.MethodGet,
			target: "/api/user/orders",
			body:   "",
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:   "order get balance #1",
			method: http.MethodGet,
			target: "/api/user/balance",
			body:   "",
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:   "order get withdrawals #1",
			method: http.MethodGet,
			target: "/api/user/withdrawals",
			body:   "",
			want: want{
				code: http.StatusNoContent,
			},
		},
	}

	mux := globalMux
	i := 0
	setCookie := ""

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.target, strings.NewReader(test.body))
			if i > 2 {
				cookie := &http.Cookie{
					Name:  "yp_diploma_one_token",
					Value: setCookie,
					Path:  "/",
				}
				request.AddCookie(cookie)
			}

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, request)

			res := w.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			res.Body.Close()

			if i == 0 {
				cookies := res.Cookies()
				for _, cookie := range cookies {
					if cookie.Name == "yp_diploma_one_token" {
						setCookie = cookie.Value
					}
				}
			}
		})

		i++
	}
}

func TestFinally(t *testing.T) {
	if globalDB != nil {
		err := globalDB.Close()
		if err != nil {
			globalLogger.Errorw(err.Error(), "event", "закрытие базы данных")
		}
	}
}
