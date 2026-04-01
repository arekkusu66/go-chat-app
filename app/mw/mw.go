package mw

import (
	"gochat/app"
	"net/http"
	"time"
)


func RateLimiter(t time.Duration) app.Middleware {
	return func(app *app.App, next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				id, status, err := app.GetUserID(r, w)

				if err != nil {
					http.Error(w, err.Error(), status)
					return
				}

				var limiter = app.RateLimiter(id.String(), t)

				if !limiter.Allow() {
					http.Error(w, "too many requests", http.StatusTooManyRequests)
					return
				}

				next.ServeHTTP(w, r)

			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}