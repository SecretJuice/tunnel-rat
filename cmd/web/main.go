package main

import (
	"conrobb/tunnel-rat/internal/model"
	"log/slog"
	"net/http"
	"os"
)

var logger *slog.Logger = slog.New(
	slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func handleCreateTunnel(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Error("Could not parse form", "error", err.Error())
		httpError(w, http.StatusBadRequest)
		return
	}
	secret := r.FormValue("client_secret")
	logger.Debug("SECRET: " + secret)
	if !model.ValidateSecret(secret) {
		httpError(w, http.StatusUnauthorized)
		return
	}

	w.Write([]byte("asdfasfsadfsa"))

}

func handleCreateClient(w http.ResponseWriter, r *http.Request) {
	newClient := model.Client{}

	secret, err := model.CreateClient(newClient)
	if err != nil {
		logger.Error("Could not create client", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}

	w.Write([]byte(secret))
}

func mwStack(h func(http.ResponseWriter, *http.Request)) http.Handler {
	return logMw(logger, http.HandlerFunc(h))
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("POST /tunnel", mwStack(handleCreateTunnel))
	mux.Handle("POST /client", mwStack(handleCreateClient))

	logger.Info("Starting server on :8080")
	http.ListenAndServe(":8080", mux)
}
