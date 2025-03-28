package main

import (
	"conrobb/tunnel-rat/internal/db"
	"conrobb/tunnel-rat/internal/model"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
)

type application struct {
	logger   *slog.Logger
	db       *sql.DB
	sessions *scs.SessionManager
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

var tempSuperUser UserInfo = UserInfo{
	Id:       1,
	Username: "temp-superuser",
}

func (app *application) handleSignin(w http.ResponseWriter, r *http.Request) {
	app.sessions.Put(r.Context(), "user-info", tempSuperUser)
}

func (app *application) handleCreateTunnel(w http.ResponseWriter, r *http.Request) {

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

	app.logger.Debug("SECRET: " + data.Secret)
	if !model.ValidateSecret(data.Secret) {
		httpError(w, http.StatusUnauthorized)
		return
	}

	w.Write([]byte("asdfasfsadfsa"))

}

func (app *application) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	newClient := model.Client{}

	newClient, err := model.CreateClient(newClient)
	if err != nil {
		app.logger.Error("Could not create client", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}

	w.Write([]byte(newClient.Secret))
}

type UserInfo struct {
	Id       int64
	Username string
}

func main() {

	gob.Register(UserInfo{})

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	dsn := "postgres://user:password@localhost:5432/mydb?sslmode=disable"
	db, err := db.Connect(dsn, logger)
	if err != nil {
		logger.Error("Could not connect to the database", "error", err.Error())
		os.Exit(1)
	}
	sessionManager := scs.New()
	sessionManager.Store = postgresstore.New(db)
	sessionManager.Lifetime = 2 * time.Hour

	app := application{
		logger:   logger,
		db:       db,
		sessions: sessionManager,
	}

	logger.Info("Starting server on :8080")
	http.ListenAndServe(":8080", app.routes())
}

func (app *application) routes() *http.ServeMux {
	mwStack := func(h func(http.ResponseWriter, *http.Request)) http.Handler {
		return app.logMw(http.HandlerFunc(h))
	}
	anonUserMw := func(h func(http.ResponseWriter, *http.Request)) http.Handler {
		return app.logMw(
			app.sessions.LoadAndSave(
				http.HandlerFunc(h),
			),
		)
	}
	protectedUserMw := func(h func(http.ResponseWriter, *http.Request)) http.Handler {
		return app.logMw(
			app.sessions.LoadAndSave(
				app.requireAuthentication(
					http.HandlerFunc(h),
				),
			),
		)
	}

	mux := http.NewServeMux()

	mux.Handle("GET /signin", anonUserMw(app.handleSignin))

	mux.Handle("POST /tunnel", mwStack(app.handleCreateTunnel))
	mux.Handle("POST /client", protectedUserMw(app.handleCreateClient))

	return mux
}
