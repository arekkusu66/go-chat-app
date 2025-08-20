package mw

import (
	"gochat/models"
	"gochat/utils"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)


func RateLimiter(handle http.HandlerFunc, duration time.Duration) http.HandlerFunc {
	var (
		mu			sync.Mutex
		clients	 =	make(map[string]*models.Client)
	)


	go func() {
		var ticker = time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			var idsToDelete = []string{}

			for id, client := range clients {
				if time.Since(client.LastRequest) > duration {
					idsToDelete = append(idsToDelete, id)
				}
			}

			for _, id := range idsToDelete {
				delete(clients, id)
			}

			mu.Unlock()
		}
	}()


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
	
			if _, found := clients[user.ID]; !found {
				clients[user.ID] = &models.Client{Limiter: rate.NewLimiter(rate.Every(duration), 1)}
			}
	
			clients[user.ID].LastRequest = time.Now()

			var allowed = clients[user.ID].Limiter.Allow()
	
			mu.Unlock()

			if !allowed {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			
			handle.ServeHTTP(w, r)

		} else {

			handle.ServeHTTP(w, r)
		}
	})
}