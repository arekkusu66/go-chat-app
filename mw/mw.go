package mw

import (
	"gochat/db"
	"gochat/utils"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)


type rateLimiter struct {
	bucket				chan struct{}
}


func newRateLimiter(duration time.Duration) *rateLimiter {
	var rl = &rateLimiter{
		bucket: make(chan struct{}, 1),
	}

	rl.bucket <- struct{}{}

	go func() {
		var ticker = time.NewTicker(duration)
		defer ticker.Stop()

		for {
			<-ticker.C
			select {
				case rl.bucket <- struct{}{}:
				default:
			}
		}
	}()

	return rl
}


func RateLimiter(handle http.HandlerFunc, duration time.Duration) http.HandlerFunc {
	var (
		mu			sync.Mutex
		clients	 =	make(map[uuid.UUID]*rateLimiter)
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			id, status, err := utils.GetUserID(r)

			if err != nil {
				http.Error(w, err.Error(), status)
				return
			}

			user, _ := db.Query.GetUserById(r.Context(), id)
	
			mu.Lock()
	
			rl, found := clients[user.ID]

			if !found {
				rl = newRateLimiter(duration)
				clients[user.ID] = rl
			}
	
			mu.Unlock()

			select {
				case <-rl.bucket:
					handle.ServeHTTP(w, r)
					return
				default:
					w.WriteHeader(http.StatusTooManyRequests)
					return
			}

		} else {
			handle.ServeHTTP(w, r)
		}
	})
}