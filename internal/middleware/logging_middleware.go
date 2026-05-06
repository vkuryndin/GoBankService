package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func RequestLogger(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := newStatusRecorder(w)

			next.ServeHTTP(recorder, r)

			duration := time.Since(startedAt)

			entry := logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      recorder.statusCode,
				"duration_ms": duration.Milliseconds(),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			})

			if recorder.statusCode >= 500 {
				entry.Error("http request completed")
				return
			}

			if recorder.statusCode >= 400 {
				entry.Warn("http request completed")
				return
			}

			entry.Info("http request completed")
		})
	}
}
