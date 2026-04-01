package app

import (
	"net/http"
)

type AppHandler func(app *App) http.HandlerFunc

type Middleware func(app *App, next http.HandlerFunc) http.HandlerFunc

func (app *App) HandleFunc(pattern string, handler AppHandler, middlewares ...Middleware) {
	var h = handler(app)

	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](app, h)
	}

	app.Mux.HandleFunc(pattern, h)
}