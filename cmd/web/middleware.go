package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type loggerWriter struct {
	http.ResponseWriter
	code *int
}

func (w *loggerWriter) WriteHeader(statusCode int) {
	*w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func logMw(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var status_code int
		loggerWriter := loggerWriter{ResponseWriter: w, code: &status_code}

		start := time.Now()

		next.ServeHTTP(&loggerWriter, r)

		elapsed := time.Since(start)

		if status_code == 0 {
			status_code = 200
		}

		logger.LogAttrs(context.Background(), slog.LevelInfo, "Request Received",
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status_code", *loggerWriter.code),
			slog.String("duration", fmt.Sprintf("%dms", elapsed.Milliseconds())),
		)
	})
}
