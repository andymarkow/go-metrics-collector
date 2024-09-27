package middlewares

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

// loggerResponseWriter wraps http.ResponseWriter and tracks the response size and status code.
// Uses in Logger middleware.
type loggerResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (w *loggerResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseData.size += size

	return size, err //nolint:wrapcheck
}

func (w *loggerResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.status = statusCode
}

// Logger is a router middleware that logs requests and their processing time.
func (m *Middlewares) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		responseData := &responseData{
			status: 200,
			size:   0,
		}

		cw := loggerResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		defer func() {
			m.log.Info("request",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Int("status", responseData.status),
				zap.Int("size", responseData.size),
				zap.Duration("duration_ms", time.Since(startTime)*1000),
			)
		}()

		next.ServeHTTP(&cw, r)
	})
}
