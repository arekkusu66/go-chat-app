package routes

import (
	"gochat/app"
	"gochat/app/db"
	"gochat/app/pages"
	"gochat/app/types"

	"net/http"
)


func SettingsH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				app.Log.Error("couldnt parse the datas", "error_info", err)
				http.Error(w, "couldnt parse the datas", http.StatusInternalServerError)
				return
			}

			var settings = r.PostForm

			if err := app.Query.UpdateSettings(r.Context(), db.UpdateSettingsParams{
				UserID: id,
				AcceptsFriendReqs: settings.Get(string(types.ACCEPTS_FRIEND_REQS)),
				AcceptsDmReqs: settings.Get(string(types.ACCEPTS_DM_REQS)),
			}); err != nil {
				app.Log.Error("couldnt update the settings", "error_info", err)
				http.Error(w, "couldnt update the settings", http.StatusInternalServerError)
				return
			}
		}

		user, _ := app.Query.GetUserById(r.Context(), id)

		pages.UserSettings(app, user).Render(r.Context(), w)
	})
}