package routes

import (
	"database/sql"
	"gochat/db"
	"gochat/pages"
	"gochat/utils"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/discord"
	goog "github.com/markbates/goth/providers/google"
)


var store *sessions.CookieStore

func InitOauth() {
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	store.Options = &sessions.Options{
    	Path:     "/",
    	MaxAge:   86400 * 7,
    	HttpOnly: true,
    	SameSite: http.SameSiteLaxMode,
	}

	gothic.Store = store

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


func OauthSignUpH(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}


func OauthCallbackH(w http.ResponseWriter, r *http.Request) {
	datas, err := gothic.CompleteUserAuth(w, r)

	if err != nil {
		http.Error(w, "couldnt fetch user datas", http.StatusInternalServerError)
		return
	}

	user, err := db.Query.GetUserByProviderId(r.Context(), sql.NullString{String: datas.UserID, Valid: true})

	var existingUserId = user.ID

	if err != nil && err == sql.ErrNoRows {
		var id = uuid.New()

		if err := db.CreateUser(r.Context(), &db.CreateUserParams{
			ID: id,
			ProviderID: sql.NullString{String: datas.UserID, Valid: true},
			Username: "user_" + id.String(),
			Email: datas.Email,
			Verified: true,
		}); err != nil {
			log.Println(utils.GetFuncInfo(), err)
			http.Error(w, "couldnt log you in", http.StatusInternalServerError)
			return
		}

		existingUserId = id

	} else {
		log.Println(utils.GetFuncInfo(), err)
		http.Error(w, "error checking for the user datas", http.StatusInternalServerError)
		return
	}

	jwtToken, err := utils.NewJWT(existingUserId)

	if err != nil {
		log.Println("error saving the jwt token", err)
		http.Error(w, "error saving the user datas", http.StatusInternalServerError)
		return
	}

	var cookie = &http.Cookie{
		Name: "token",
		Value: jwtToken,
		Path: "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
	
	pages.OauthCreds().Render(r.Context(), w)
}