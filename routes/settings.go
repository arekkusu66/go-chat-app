package routes

import (
	"gochat/db"
	"gochat/pages"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
)


func SettingsH(w http.ResponseWriter, r *http.Request) {
	id, status, err := utils.GetUserID(r, w)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			log.Println(err)
			http.Error(w, "couldnt parse the datas", http.StatusInternalServerError)
			return
		}

		var settings = r.PostForm

		if err := db.Query.UpdateSettings(r.Context(), db.UpdateSettingsParams{
			UserID: id,
			AcceptsFriendReqs: settings.Get(string(types.ACCEPTS_FRIEND_REQS)),
			AcceptsDmReqs: settings.Get(string(types.ACCEPTS_DM_REQS)),
		}); err != nil {
			http.Error(w, "couldnt update the settings", http.StatusInternalServerError)
			return
		}
	}

	user, _ := db.Query.GetUserById(r.Context(), id)

	pages.UserSettings(user).Render(r.Context(), w)
}