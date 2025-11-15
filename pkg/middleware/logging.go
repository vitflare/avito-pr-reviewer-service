package middleware

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status >= http.StatusBadRequest {
		rw.body.Write(b)
	}

	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			body:           &bytes.Buffer{},
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		logMsg := fmt.Sprintf(
			"%s %s - %d %dB in %s",
			r.Method,
			r.RequestURI,
			rw.status,
			rw.size,
			duration.String(),
		)

		if rw.status >= http.StatusBadRequest {
			slog.Error(logMsg,
				"response_body", rw.body.String(),
			)
		} else {
			slog.Info(logMsg)
		}
	})
}
