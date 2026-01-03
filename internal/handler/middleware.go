// Package handler содержит в себе типы данных и middleware для обработки HTTP запросов
package handler

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"database/sql"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/hardvlad/ypdiploma1/internal/auth"
	"go.uber.org/zap"
)

type contextKey string

// UserIDKey поле в контексте запроса для UserID
const userIDKey contextKey = "user_id"

type compressWriter struct {
	http.ResponseWriter
	Writer        io.Writer
	setEncoding   string
	setStatusCode int
}

func (w *compressWriter) Write(b []byte) (int, error) {
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		w.Header().Set("Content-Encoding", w.setEncoding)
		w.ResponseWriter.WriteHeader(w.setStatusCode)
		return w.Writer.Write(b)
	}
	w.ResponseWriter.WriteHeader(w.setStatusCode)
	return w.ResponseWriter.Write(b)
}

func (w *compressWriter) WriteHeader(statusCode int) {
	w.setStatusCode = statusCode
}

// ResponseCompressHandle возвращает хендлер middleware для сжатия ответов
func ResponseCompressHandle(next http.Handler, sugarLogger *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding := r.Header.Get("Accept-Encoding")
		var writer io.WriteCloser
		var err error = nil
		encoding := ""

		if slices.Contains([]string{"br", "gzip", "deflate"}, acceptEncoding) {
			encoding = acceptEncoding
			switch acceptEncoding {
			case "br":
				writer = brotli.NewWriterLevel(w, brotli.BestCompression)
			case "gzip":
				writer, err = gzip.NewWriterLevel(w, gzip.BestCompression)
			case "deflate":
				writer, err = zlib.NewWriterLevel(w, flate.BestCompression)
			}

			if err != nil {
				next.ServeHTTP(w, r)
				sugarLogger.Error(err.Error(), "сжатие ответа", acceptEncoding)
				return
			}

			defer writer.Close()
		} else {
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(&compressWriter{ResponseWriter: w, Writer: writer, setEncoding: encoding, setStatusCode: 0}, r)
	})
}

// RequestDecompressHandle возвращает хендлер middleware для декомпрессии запросов
func RequestDecompressHandle(next http.Handler, sugarLogger *zap.SugaredLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var reader io.ReadCloser
		var err error

		contentEncoding := r.Header.Get("Content-Encoding")

		if contentEncoding == `` {
			next.ServeHTTP(w, r)
			return
		}

		if contentEncoding == `gzip` {
			reader, err = gzip.NewReader(r.Body)
		} else if contentEncoding == `br` {
			reader = io.NopCloser(brotli.NewReader(r.Body))
		} else if contentEncoding == `deflate` {
			reader = flate.NewReader(r.Body)
		} else if contentEncoding != `` {
			http.Error(w, "decompressor not found", http.StatusInternalServerError)
			return
		}

		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			sugarLogger.Error(err.Error(), "распаковка запроса", contentEncoding)
			return
		}

		r.Body = reader
		defer reader.Close()
		r.Header.Del("Content-Encoding")
		next.ServeHTTP(w, r)
	})
}

// AuthorizationMiddleware возвращает хендлер middleware для проверки авторизации
func AuthorizationMiddleware(next http.Handler, sugarLogger *zap.SugaredLogger, cookieName string, secretKey string, db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// пути, требующие авторизации
		authRoutes := []string{"/api/user/orders", "/api/user/balance", "/api/user/balance/withdraw", "/api/user/withdrawals"}
		// если авторизация не нужна - пропускаем обработку
		if !slices.Contains(authRoutes, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// если не заданы параметры авторизации - выдаем StatusUnauthorized
		if cookieName == "" || secretKey == "" || db == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		userID := 0
		c, err := r.Cookie(cookieName)
		if err != nil {
		} else {
			uid, err := auth.GetUserID(c.Value, secretKey)
			// если возникла ошибка при разборе JWT токена - выдаем StatusUnauthorized
			if err != nil {
				sugarLogger.Errorw(err.Error(), "event", "парсинг токена из куки", "cookie", c.Value)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID = uid
		}

		// если удалось определить ID пользователя - выдаем StatusUnauthorized
		if userID == 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserIDFromRequest(r *http.Request) (userID int, ok bool) {
	userID, ok = r.Context().Value(userIDKey).(int)
	return userID, ok
}
