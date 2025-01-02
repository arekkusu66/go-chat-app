package mw

import (
	"gochat/models"
	"gochat/routes"
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
		for {
			for email, c := range clients {
				mu.Lock()
				
				if time.Since(c.LastRequest) > duration {
					delete(clients, email)
				}
	
				mu.Unlock()
			}
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
	
			mu.Lock()
	
			models.DB.First(&user, "id = ?", userData.ID)
	
			if _, found := clients[user.ID]; !found {
				clients[user.ID] = &models.Client{Limiter: rate.NewLimiter(rate.Every(duration), 1)}
			}
	
			clients[user.ID].LastRequest = time.Now()
	
			if !clients[user.ID].Limiter.Allow() {
				mu.Unlock()
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
	
			mu.Unlock()
			
			handle.ServeHTTP(w, r)

		} else {

			handle.ServeHTTP(w, r)
		}
	})
}


func Ws(wsconf models.WSconf) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch wsconf.Type {
			case "chat":
				routes.MSGWS(w, r, wsconf.Clients, true)
			case "dm":
				routes.MSGWS(w, r, wsconf.Clients, false)
			case "del":
				routes.DELWS(w, r, wsconf.Clients)
			case "notif":
				routes.NOTIFWS(w, r, wsconf.Clients)
		}
	})
}