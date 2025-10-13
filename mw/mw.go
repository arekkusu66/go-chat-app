package mw

import (
	"gochat/models"
	"gochat/utils"
	"net/http"
	"sync"
	"time"
)


func newRateLimiter(duration time.Duration) *models.RateLimiter {
	var rl = &models.RateLimiter{
		Bucket: make(chan struct{}, 1),
	}

	rl.Bucket <- struct{}{}

	go func() {
		var ticker = time.NewTicker(duration)
		defer ticker.Stop()

		for {
			<-ticker.C
			select {
				case rl.Bucket <- struct{}{}:
				default:
			}
		}
	}()

	return rl
}


func RateLimiter(handle http.HandlerFunc, duration time.Duration) http.HandlerFunc {
	var (
		mu			sync.Mutex
		clients	 =	make(map[string]*models.RateLimiter)
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			
			var user models.User
			userData, err := utils.ParseCookie(r)
	
			if err != nil {
				http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
				return
			}

			models.DB.First(&user, "id = ?", userData.ID)
	
			mu.Lock()
	
			rl, found := clients[user.ID]

			if !found {
				rl = newRateLimiter(duration)
				clients[user.ID] = rl
			}
	
			mu.Unlock()

			select {
				case <-rl.Bucket:
					handle.ServeHTTP(w, r)
					return
				default:
					http.Error(w, "too many requests!", http.StatusTooManyRequests)
					return
			}

		} else {
			handle.ServeHTTP(w, r)
		}
	})
}