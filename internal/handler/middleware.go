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

const UserIDKey contextKey = "user_id"

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

func AuthorizationMiddleware(next http.Handler, sugarLogger *zap.SugaredLogger, cookieName string, secretKey string, db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if cookieName == "" || secretKey == "" || db == nil {
			next.ServeHTTP(w, r)
			return
		}

		userID := 0
		c, err := r.Cookie(cookieName)
		if err != nil {
		} else {
			uid, err := auth.GetUserID(c.Value, secretKey)
			if err != nil {
				sugarLogger.Errorw(err.Error(), "event", "парсинг токена из куки", "cookie", c.Value)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID = uid
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
