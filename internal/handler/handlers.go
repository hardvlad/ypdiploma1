package handler

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/hardvlad/ypdiploma1/internal/config"
	"github.com/hardvlad/ypdiploma1/internal/repository"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

type DeleteChannelRequest struct {
	UserID int
	URLs   []string
}

type Handlers struct {
	Conf   *config.Config
	Store  repository.StorageInterface
	Logger *zap.SugaredLogger
}

type shortenerResponse struct {
	isError     bool
	message     string
	redirectURL string
	code        int
}

type URLRequest struct {
	URL string `json:"url"`
}

type BatchURLRequest struct {
	ID  string `json:"correlation_id"`
	URL string `json:"original_url"`
}

type BatchURLResponseObject struct {
	ID  string `json:"correlation_id"`
	URL string `json:"short_url"`
}

func NewHandlers(conf *config.Config, store repository.StorageInterface, sugarLogger *zap.SugaredLogger) http.Handler {

	mux := chi.NewRouter()

	handlersData := Handlers{
		Conf:   conf,
		Store:  store,
		Logger: sugarLogger,
	}

	ch := make(chan DeleteChannelRequest, 100)
	go deleteWorker(handlersData, ch)

	mux.Post(`/api/user/register`, createPostHandler(handlersData))
	mux.Post(`/api/user/login`, createPostHandler(handlersData))
	mux.Post(`/api/user/orders`, createPostHandler(handlersData))

	mux.Get(`/api/user/orders`, createGetHandler(handlersData))
	mux.Get(`/api/user/balance`, createGetHandler(handlersData))

	mux.Post(`/api/user/balance/withdraw`, createPostHandler(handlersData))
	mux.Get(`/api/user/withdrawals`, createGetHandler(handlersData))

	return mux
}

func createPostHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "can't read body",
				code:    http.StatusBadRequest,
			})
			return
		}

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			userID = 0
		}

		writeResponse(w, r, processNewURL(data, string(bodyBytes), userID))
	}
}

func createGetHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeResponse(w, r, processRedirect(data, chi.URLParam(r, "code")))
	}
}

func createPostJSONHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req URLRequest
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "can't decode JSON",
				code:    http.StatusBadRequest,
			})
			return
		}

		if req.URL == "" {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "please post URL in JSON",
				code:    http.StatusBadRequest,
			})
			return
		}

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			userID = 0
		}

		resp := processNewURL(data, req.URL, userID)
		if resp.isError {
			writeResponse(w, r, resp)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.code)
			json.NewEncoder(w).Encode(map[string]string{"result": resp.message})
		}
	}
}

func createPostJSONBatchHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resp []BatchURLResponseObject

		var req []BatchURLRequest
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "can't decode JSON",
				code:    http.StatusBadRequest,
			})
			return
		}

		if len(req) == 0 {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "please post correct JSON",
				code:    http.StatusBadRequest,
			})
			return
		}

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			userID = 0
		}

		for _, urlData := range req {
			success, shortLink, _, err := GetShortCode(data, urlData.URL, 5, userID)
			if err != nil {
				data.Logger.Debugw(err.Error(), "event", "добавление URL", "url", urlData.URL)
			}

			if success {
				fullURL, _ := url.JoinPath(data.Conf.ServerAddress, shortLink)
				resp = append(resp, BatchURLResponseObject{ID: urlData.ID, URL: fullURL})
			}
		}

		if len(resp) == 0 {
			writeResponse(w, r, shortenerResponse{
				isError: false,
				message: "no data in response",
				code:    http.StatusBadRequest,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func createPingDBHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeResponse(w, r, pingDB(data))
	}
}

func createGetUserURLSHandler(data Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			userID = 0
		}

		userURLs, err := data.Store.GetUserData(userID)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "получение данных пользователя", "user_id", userID)
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: http.StatusText(http.StatusInternalServerError),
				code:    http.StatusInternalServerError,
			})
			return
		}

		if len(userURLs) == 0 {
			writeResponse(w, r, shortenerResponse{
				isError: false,
				message: "no URLs for this user",
				code:    http.StatusNoContent,
			})
			return
		}

		type UserURLResponse struct {
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
		}

		var resp []UserURLResponse
		for shortCode, originalURL := range userURLs {
			fullURL, _ := url.JoinPath(data.Conf.ServerAddress, shortCode)
			resp = append(resp, UserURLResponse{
				ShortURL:    fullURL,
				OriginalURL: originalURL,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func createDeleteUserURLSHandler(data Handlers, ch chan DeleteChannelRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var urlsToDelete []string
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&urlsToDelete); err != nil {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "can't decode JSON",
				code:    http.StatusBadRequest,
			})
			return
		}

		if len(urlsToDelete) == 0 {
			writeResponse(w, r, shortenerResponse{
				isError: true,
				message: "please post correct JSON",
				code:    http.StatusBadRequest,
			})
			return
		}

		userID, ok := r.Context().Value(UserIDKey).(int)
		if !ok {
			userID = 0
		}

		ch <- DeleteChannelRequest{
			UserID: userID,
			URLs:   urlsToDelete,
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func deleteWorker(data Handlers, ch chan DeleteChannelRequest) {
	for req := range ch {
		err := data.Store.DeleteURLs(req.URLs, req.UserID)
		if err != nil {
			data.Logger.Debugw(err.Error(), "event", "удаление URL", "user_id", req.UserID, "urls", req.URLs)
		}
	}
}

func writeResponse(w http.ResponseWriter, r *http.Request, resp shortenerResponse) {
	if resp.isError {
		http.Error(w, resp.message, resp.code)
	} else {
		if resp.redirectURL != "" {
			http.Redirect(w, r, resp.redirectURL, resp.code)
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(resp.code)
			_, err := w.Write([]byte(resp.message))
			if err != nil {
				return
			}
		}
	}
}

func pingDB(data Handlers) shortenerResponse {
	database, err := data.Conf.DBConfig.InitDB()
	if err != nil {
		if data.Logger != nil {
			data.Logger.Errorw(err.Error(), "event", "соединение с базой данных")
		}
		return shortenerResponse{
			isError: false,
			message: http.StatusText(http.StatusInternalServerError),
			code:    http.StatusInternalServerError,
		}
	}

	database.Close()

	return shortenerResponse{
		isError: false,
		message: http.StatusText(http.StatusOK),
		code:    http.StatusOK,
	}
}

func processRedirect(data Handlers, path string) shortenerResponse {
	urlRedirect, isDeleted, ok := data.Store.Get(path)
	if isDeleted {
		return shortenerResponse{
			isError: true,
			message: "short link was deleted",
			code:    http.StatusGone,
		}
	}

	if ok {
		return shortenerResponse{
			isError:     false,
			redirectURL: urlRedirect,
			code:        http.StatusTemporaryRedirect,
		}
	}

	return shortenerResponse{
		isError: true,
		message: "short link does not exist",
		code:    http.StatusBadRequest,
	}
}

func processNewURL(data Handlers, body string, userID int) shortenerResponse {

	success, shortLink, urlAlreadyExisted, err := GetShortCode(data, body, 5, userID)
	if err != nil {
		data.Logger.Debugw(err.Error(), "event", "добавление URL", "url", body)
	}

	if !success {
		return shortenerResponse{
			isError: true,
			message: http.StatusText(http.StatusInternalServerError),
			code:    http.StatusInternalServerError,
		}
	}

	fullURL, err := url.JoinPath(data.Conf.ServerAddress, shortLink)
	if err != nil {
		return shortenerResponse{
			isError: true,
			message: http.StatusText(http.StatusInternalServerError),
			code:    http.StatusInternalServerError,
		}
	}

	if urlAlreadyExisted {
		return shortenerResponse{
			isError: false,
			message: fullURL,
			code:    http.StatusConflict,
		}
	}

	return shortenerResponse{
		isError: false,
		message: fullURL,
		code:    http.StatusCreated,
	}
}

func GenerateRandomString(conf *config.Config) string {
	b := make([]byte, conf.ShortLinkLength)
	for i := 0; i < conf.ShortLinkLength; i++ {
		b[i] = conf.Charset[rand.Intn(len(conf.Charset))]
	}
	return string(b[:])
}

func GetShortCode(data Handlers, body string, maxAttempts int, userID int) (success bool, code string, urlExisted bool, err error) {
	success = false
	var shortLink string
	var urlAlreadyExisted bool

	for i := 0; i < maxAttempts; i++ {
		shortLink = GenerateRandomString(data.Conf)
		code, urlExisted, err := data.Store.Set(shortLink, body, userID)
		if err != nil {
			if errors.Is(err, repository.ErrorKeyExists) {
				continue
			} else {
				return false, "", false, err
			}
		} else {
			success = true
			shortLink = code
			urlAlreadyExisted = urlExisted
			break
		}
	}
	return success, shortLink, urlAlreadyExisted, err
}
