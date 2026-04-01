package main

import (
	"context"
	"gochat/app"
	"gochat/app/mw"
	"gochat/app/routes"
	"gochat/app/utils"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var start = time.Now()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	app, err := app.InitApp()

	if err != nil {
		log.Println("couldnt initiate the app:", err)
		return
	}

	var mux = http.NewServeMux()
	app.Mux = mux

	utils.ServeDir("/script", mux)

	app.HandleFunc("/{$}", routes.HomeH)
	app.HandleFunc("/chat/{id}", routes.ChatH)
	app.HandleFunc("POST /chat/actions", routes.ChatActionsH)
	app.HandleFunc("POST /create/chat", routes.CreateChatH, mw.RateLimiter(time.Minute))
	app.HandleFunc("/dms", routes.DMchatroomH)
	app.HandleFunc("/dm/{id}", routes.DMH)
	app.HandleFunc("POST /dm/action", routes.DMactionH)
	app.HandleFunc("POST /get/options", routes.GetOptionsH)
	app.HandleFunc("POST /get/message", routes.GetMessageH)
	app.HandleFunc("POST /notifications", routes.NotificationsH)

	app.HandleFunc("/user/{username}", routes.UserPageH)
	app.HandleFunc("POST /edit/username", routes.EditUsernameH, mw.RateLimiter(time.Hour * 24 * 14))
	app.HandleFunc("POST /edit/profile", routes.EditProfileH)
	app.HandleFunc("POST /user/actions", routes.UserActionsH)
	app.HandleFunc("/settings", routes.SettingsH)

	app.HandleFunc("/signup", routes.SignUpH)
	app.HandleFunc("/login", routes.LoginH)
	app.HandleFunc("/oauth/signup/{provider}", routes.OauthSignUpH)
	app.HandleFunc("/oauth/creds/{provider}", routes.OauthCallbackH)
	app.HandleFunc("POST /logoff", routes.LogOffH)
	app.HandleFunc("/email/verification", routes.EmailVerificationH)
	app.HandleFunc("/email/verification/send", routes.EmailVerificationSendH, mw.RateLimiter(time.Minute))
	app.HandleFunc("/password/reset", routes.PasswordResetH, mw.RateLimiter(time.Minute))
	app.HandleFunc("/password/new", routes.PasswordNewH, mw.RateLimiter(time.Minute))

	app.HandleFunc("/ws/{kind}/{id}", routes.WSHandler)


	var srv = &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	go srv.ListenAndServe()

	app.Log.Info("app is up and running", "pid", syscall.Getpid())

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	srv.Shutdown(shutdownCtx)

	app.Log.Info("app shutted down", "uptime", time.Since(start))
	app.Close()
}