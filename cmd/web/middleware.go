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

func (app *application) logMw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var status_code int
		loggerWriter := loggerWriter{ResponseWriter: w, code: &status_code}

		start := time.Now()

		next.ServeHTTP(&loggerWriter, r)

		elapsed := time.Since(start)

		if status_code == 0 {
			status_code = 200
		}

		app.logger.LogAttrs(context.Background(), slog.LevelInfo, "Request Received",
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status_code", *loggerWriter.code),
			slog.String("duration", fmt.Sprintf("%dms", elapsed.Milliseconds())),
		)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userInfo, ok := app.getUserInfo(r)
		if !ok {
			// TODO create real auth redirect. For now, just bounce the connection
			app.logger.Debug("BOUNCING UNAUTHORIZED: userInfo NOT PRESENT")
			httpError(w, http.StatusUnauthorized)
			return
		}

		// TODO temp super user
		if userInfo.Id != 1 {
			app.logger.Debug("BOUNCING UNAUTHORIZED: INCORRECT USER ID")
			httpError(w, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) getUserInfo(r *http.Request) (UserInfo, bool) {
	userInfo, ok := app.sessions.Get(r.Context(), "user-info").(UserInfo)
	if !ok {
		return UserInfo{}, false
	}
	return userInfo, true
}
