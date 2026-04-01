package app

import (
	"database/sql"
	"gochat/app/db"

	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)


type App struct {
	DB               *sql.DB
	Query			 *db.Queries
	NcConn			 *nats.Conn
	Log              *slog.Logger
	LogFile			 *os.File
	Store			 *sessions.CookieStore
	RateLimitClients sync.Map
	Mux              *http.ServeMux
}


func InitApp() (*App, error) {
	var app = &App{}

	var err error


	// loading environment
	if err = godotenv.Load(); err != nil {
		log.Println("couldnt read the environment variables")
		return nil, err
	}


	// initiating the database
	db, query, err := db.InitDB()

	if err != nil {
		log.Println("could initiate a connection to the database")
		return nil, err
	}

	app.DB = db
	app.Query = query

	defer func() {
		if err != nil && db != nil {
			db.Close()
		}
	}()


	// initiate the nats connection
	nc, err := nats.Connect(nats.DefaultURL)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	app.NcConn = nc

	defer func() {
		if err != nil && nc != nil {
			nc.Drain()
		}
	}()


	// initiating the logger
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		return nil, err
	}

	app.LogFile = logFile
	app.Log = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{AddSource: true}))


	// initiating the oauth store and providers
	InitOauth(app)


	return app, nil
}


func (app *App) Close() {
	app.DB.Close()
	app.LogFile.Close()
}