package main

import (
	"conrobb/tunnel-rat/internal/model"
	"encoding/json"
	"io"
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

	type createTunnelReq struct {
		Secret    string `json:"client_secret"`
		PublicKey string `json:"public_key"`
	}

	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var data createTunnelReq
	if err := json.Unmarshal(body, &data); err != nil {
		httpError(w, http.StatusBadRequest)
		return
	}

	logger.Debug("SECRET: " + data.Secret)
	if !model.ValidateSecret(data.Secret) {
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
