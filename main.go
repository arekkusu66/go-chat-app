package main

import (
	"gochat/models"
	"gochat/mw"
	"gochat/routes"
	"log"
	"net/http"
	"time"

	"github.com/arekkusu66/goutils/serve"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)


func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
		return
	}

	if err := models.InitDB(); err != nil {
		log.Fatal(err)
		return
	}
}


func main() {
	var mux = http.NewServeMux()

	serve.Dir("/script", mux)


	mux.HandleFunc("/{$}", routes.HomeH)
	mux.HandleFunc("/chat/{id}", routes.ChatH)
	mux.HandleFunc("/dms", routes.DMchatroomH)
	mux.HandleFunc("/dm/{id}", routes.DMH)
	mux.HandleFunc("POST /dm/action", routes.DMactionH)
	mux.HandleFunc("POST /create/chat", mw.RateLimiter(routes.CreateChatH, time.Minute))
	mux.HandleFunc("POST /get/options", routes.GetOptionsH)
	mux.HandleFunc("POST /get/datas", routes.GetDatasH)
	mux.HandleFunc("POST /get/message", routes.GetMessageH)
	mux.HandleFunc("POST /join/chat/{id}", routes.JoinChatH)
	mux.HandleFunc("POST /leave/chat/{id}", routes.LeaveChatH)
	mux.HandleFunc("POST /notifications", routes.NotificationsH)

	mux.HandleFunc("/user/{username}", routes.UserPageH)
	mux.HandleFunc("POST /edit/username", mw.RateLimiter(routes.EditUsernameH, time.Hour * 24 * 14))
	mux.HandleFunc("POST /exists/{username}", routes.CheckAvailabilityH)
	mux.HandleFunc("POST /edit/profile", routes.EditProfileH)
	mux.HandleFunc("POST /user/actions", routes.UserActionsH)
	mux.HandleFunc("/settings", routes.SettingsH)

	mux.HandleFunc("/signup", routes.SignUpH)
	mux.HandleFunc("/login", routes.LoginH)
	mux.HandleFunc("/oauth/signup", routes.OauthSignUpH)
	mux.HandleFunc("/oauth/creds", routes.OauthCredsH)
	mux.HandleFunc("POST /logoff", routes.LogOffH)
	mux.HandleFunc("/email/verification", routes.EmailVerificationH)
	mux.HandleFunc("/email/verification/send", mw.RateLimiter(routes.EmailVerificationSendH, time.Minute))
	mux.HandleFunc("/password/reset", mw.RateLimiter(routes.PasswordResetH, time.Minute))
	mux.HandleFunc("/password/new", mw.RateLimiter(routes.PasswordNewH, time.Minute))

	mux.HandleFunc("/msg/ws/{id}", mw.Ws(models.WSconf{Type: "chat", Clients: make(map[*websocket.Conn]string)}))
	mux.HandleFunc("/dm/ws/{id}", mw.Ws(models.WSconf{Type: "dm", Clients: make(map[*websocket.Conn]string)}))
	mux.HandleFunc("/msg/ws/del/{id}", mw.Ws(models.WSconf{Type: "del", Clients: make(map[*websocket.Conn]string)}))
	mux.HandleFunc("/dm/ws/del/{id}", mw.Ws(models.WSconf{Type: "del", Clients: make(map[*websocket.Conn]string)}))
	mux.HandleFunc("/notif/ws", mw.Ws(models.WSconf{Type: "notif", Clients: make(map[*websocket.Conn]string)}))


	log.Fatal(http.ListenAndServe(":5173", mux))
}