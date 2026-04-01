package app

import (
	"os"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/discord"
	goog "github.com/markbates/goth/providers/google"
)


func InitOauth(app *App) {
	app.Store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	app.Store.Options = &sessions.Options{
    	Path:     "/",
    	MaxAge:   86400 * 7,
    	HttpOnly: true,
	}

	gothic.Store = app.Store

	goth.UseProviders(
		goog.New(
			os.Getenv("GOOGLE_CLIENT_ID"), 
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_REDIRECT_URL"),
			"email", "profile",
		),

		discord.New(
			os.Getenv("DISCORD_CLIENT_ID"),
			os.Getenv("DISCORD_CLIENT_SECRET"),
			os.Getenv("DISCORD_REDIRECT_URL"),
			"email", "identify",
		),
	)
}