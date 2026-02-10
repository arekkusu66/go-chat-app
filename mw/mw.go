package mw

import (
	"gochat/utils"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)


var clients sync.Map


func newRateLimiter(id string, t time.Duration) *rate.Limiter {
	if limiter, ok := clients.Load(id); ok {
		return limiter.(*rate.Limiter)
	}

	var limiter = rate.NewLimiter(rate.Every(t), 1)

	actual, _ := clients.LoadOrStore(id, limiter)

	return actual.(*rate.Limiter)
}


func RateLimiter(handle http.HandlerFunc, t time.Duration) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			id, status, err := utils.GetUserID(r, w)

			if err != nil {
				http.Error(w, err.Error(), status)
				return
			}

			var limiter = newRateLimiter(id.String(), t)

			if !limiter.Allow() {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			handle.ServeHTTP(w, r)

		} else {
			handle.ServeHTTP(w, r)
		}
	})
}