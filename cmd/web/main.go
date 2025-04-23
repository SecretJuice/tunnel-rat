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
	"strconv"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
)

type application struct {
	logger   *slog.Logger
	db       *sql.DB
	sessions *scs.SessionManager
	models   *model.ModelContext
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
		PublicKey string `json:"public_key"`
	}
	client, ok := getClient(r)
	if !ok {
		httpError(w, http.StatusUnauthorized)
		return
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

	err = app.models.Clients.UpdatePublicKey(client.ID, data.PublicKey)
	if err != nil {
		app.logger.Error("Could not update public key", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}

	tunnelId, err := app.models.Tunnels.CreateTunnel(client.ID, client.Relay)
	if err != nil {
		app.logger.Error("Could not create tunnel", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}

	res := map[string]int64{
		"tunnel_id": tunnelId,
	}

	jsonBytes, err := json.Marshal(res)
	if err != nil {
		app.logger.Error("Could not marshal JSON", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}

	w.Write(jsonBytes)
}

func (app *application) handleGetTunnel(w http.ResponseWriter, r *http.Request) {
	client, ok := getClient(r)
	if !ok {
		httpError(w, http.StatusUnauthorized)
		return
	}

	tunnelID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpError(w, http.StatusBadRequest)
		return
	}

	tunnel, err := app.models.Tunnels.GetById(tunnelID)
	if err != nil {
		app.logger.Error("Could not get tunnel", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}
	if tunnel == nil {
		httpError(w, http.StatusNotFound)
		return
	}

	if tunnel.ClientID != client.ID {
		httpError(w, http.StatusForbidden)
		return
	}

	jsonBytes, err := json.Marshal(*tunnel)
	if err != nil {
		app.logger.Error("Could not marshal JSON", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}
	w.Write(jsonBytes)
}

func (app *application) handleCreateClient(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		httpError(w, http.StatusBadRequest)
		return
	}

	userInfo, ok := app.getUserInfo(r)
	if !ok {
		httpError(w, http.StatusUnauthorized)
		return
	}

	newClient, err := app.models.Clients.CreateClient(
		userInfo.Id, r.FormValue("name"), model.Region(r.FormValue("region")))
	if err != nil {
		app.logger.Error("Could not create client", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(newClient); err != nil {
		app.logger.Error("Failed to encode JSON", "error", err.Error())
		httpError(w, http.StatusInternalServerError)
		return
	}
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

	mc := initializeModelContext(db, logger)

	app := application{
		logger:   logger,
		db:       db,
		sessions: sessionManager,
		models:   mc,
	}

	logger.Info("Starting server on :8080")
	http.ListenAndServe(":8080", app.routes())
}

func initializeModelContext(db *sql.DB, logger *slog.Logger) *model.ModelContext {
	mc := model.ModelContext{Db: db, Logger: logger}

	clients := model.ClientModel{&mc}
	relays := model.RelayModel{&mc}
	tunnel := model.TunnelModel{&mc}

	mc.Clients = clients
	mc.Relays = relays
	mc.Tunnels = tunnel

	return &mc
}

func (app *application) routes() *http.ServeMux {
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
	protectedClientMw := func(h func(http.ResponseWriter, *http.Request)) http.Handler {
		return app.logMw(
			app.clientRequireAuth(
				http.HandlerFunc(h),
			),
		)
	}

	mux := http.NewServeMux()

	mux.Handle("GET /signin", anonUserMw(app.handleSignin))

	mux.Handle("POST /tunnel", protectedClientMw(app.handleCreateTunnel))
	mux.Handle("GET /tunnel/{id}", protectedClientMw(app.handleGetTunnel))

	mux.Handle("POST /client", protectedUserMw(app.handleCreateClient))

	return mux
}
