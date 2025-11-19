package main

import (
	"gochat/db"
	"gochat/mw"
	"gochat/routes"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)


func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("couldnt load the env", err)
	}

	if err := db.InitDB(); err != nil {
		log.Fatal("couldnt init the database", err)
	}

	routes.InitOauth()
}


func main() {
	var (
		mux = http.NewServeMux()
		ws = routes.NewHub()
	)

	go ws.Run()
	utils.ServeDir("/script", mux)

	mux.HandleFunc("/{$}", routes.HomeH)
	mux.HandleFunc("/chat/{id}", routes.ChatH)
	mux.HandleFunc("POST /chat/actions", routes.ChatActionsH)
	mux.HandleFunc("POST /create/chat", mw.RateLimiter(routes.CreateChatH, time.Minute))
	mux.HandleFunc("/dms", routes.DMchatroomH)
	mux.HandleFunc("/dm/{id}", routes.DMH)
	mux.HandleFunc("POST /dm/action", routes.DMactionH)
	mux.HandleFunc("POST /get/options", routes.GetOptionsH)
	mux.HandleFunc("POST /get/message", routes.GetMessageH)
	mux.HandleFunc("POST /notifications", routes.NotificationsH)

	mux.HandleFunc("/user/{username}", routes.UserPageH)
	mux.HandleFunc("POST /edit/username", mw.RateLimiter(routes.EditUsernameH, time.Hour * 24 * 14))
	mux.HandleFunc("POST /edit/profile", routes.EditProfileH)
	mux.HandleFunc("POST /user/actions", routes.UserActionsH)
	mux.HandleFunc("/settings", routes.SettingsH)

	mux.HandleFunc("/signup", routes.SignUpH)
	mux.HandleFunc("/login", routes.LoginH)
	mux.HandleFunc("/oauth/signup/{provider}", routes.OauthSignUpH)
	mux.HandleFunc("/oauth/creds/{provider}", routes.OauthCallbackH)
	mux.HandleFunc("POST /logoff", routes.LogOffH)
	mux.HandleFunc("/email/verification", routes.EmailVerificationH)
	mux.HandleFunc("/email/verification/send", mw.RateLimiter(routes.EmailVerificationSendH, time.Minute))
	mux.HandleFunc("/password/reset", mw.RateLimiter(routes.PasswordResetH, time.Minute))
	mux.HandleFunc("/password/new", mw.RateLimiter(routes.PasswordNewH, time.Minute))

	mux.HandleFunc("/msg/ws/{id}", ws.WSHandler(types.MSG))
	mux.HandleFunc("/notif/ws", ws.WSHandler(types.NOTIF))


	log.Fatal(http.ListenAndServe(":8080", mux))
}