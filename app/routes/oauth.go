package routes

import (
	"database/sql"
	"gochat/app"
	"gochat/app/db"
	"gochat/app/pages"
	"net/http"

	"github.com/google/uuid"
	"github.com/markbates/goth/gothic"
)


func OauthSignUpH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gothic.BeginAuthHandler(w, r)
	})
}


func OauthCallbackH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datas, err := gothic.CompleteUserAuth(w, r)

		if err != nil {
			app.Log.Error("couldnt fetch the user datas", "error_info", err)
			http.Error(w, "couldnt fetch user datas", http.StatusInternalServerError)
			return
		}

		user, err := app.Query.GetUserByProviderId(r.Context(), sql.NullString{String: datas.UserID, Valid: true})

		var existingUserId = user.ID

		if err != nil && err == sql.ErrNoRows {
			var id = uuid.New()

			if err := db.CreateUser(r.Context(), app.DB, app.Query, &db.CreateUserParams{
				ID: id,
				ProviderID: sql.NullString{String: datas.UserID, Valid: true},
				Username: "user_" + id.String(),
				Email: datas.Email,
				Verified: true,
			}); err != nil {
				app.Log.Error("couldnt create the user", "error_info", err)
				http.Error(w, "couldnt log you in", http.StatusInternalServerError)
				return
			}

			existingUserId = id

		} else {
			app.Log.Error("couldnt check for the user datas", "error_info", err)
			http.Error(w, "error checking for the user datas", http.StatusInternalServerError)
			return
		}

		jwtToken, err := app.NewJWT(existingUserId)

		if err != nil {
			app.Log.Error("couldnt save the user datas", "error_info", err)
			http.Error(w, "error saving the user datas", http.StatusInternalServerError)
			return
		}

		var cookie = &http.Cookie{
			Name: "token",
			Value: jwtToken,
			Path: "/",
			HttpOnly: true,
		}

		http.SetCookie(w, cookie)
		
		pages.OauthCreds().Render(r.Context(), w)
	})
}