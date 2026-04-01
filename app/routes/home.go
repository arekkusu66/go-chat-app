package routes

import (
	"gochat/app"
	"gochat/app/pages"
	"net/http"
)


func HomeH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _, err := app.GetUserID(r, w)

		if err != nil {
			app.Log.Warn("couldnt get the user datas", "error_info", err)
			pages.Home(nil, nil).Render(r.Context(), w)
			return
		}

		user, err := app.Query.GetUserById(r.Context(), id)
		
		if err != nil {
			app.Log.Warn("couldnt get the user datas", "error_info", err)
			pages.Home(nil, nil).Render(r.Context(), w)
			return
		}

		pages.Home(app, &user).Render(r.Context(), w)
	})
}