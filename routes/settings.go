package routes

import (
	"gochat/models"
	"gochat/types"
	"gochat/utils"
	"net/http"
)


func SettingsH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user datas", http.StatusInternalServerError)
		return
	}

	var user models.User

	if err := models.DB.Preload("Settings").First(&user, "id = ?", userData.ID).Error; err != nil {
		http.Error(w, "couldnt retrieve user datas", http.StatusInternalServerError)
		return
	}
	
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "couldnt get the datas from the form", http.StatusInternalServerError)
			return
		}

		if err := utils.SettingMod(r, user, types.ACCEPT_FRIEND_REQ, "accepts_friend_reqs"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := utils.SettingMod(r, user, types.DM_REQ, "accepts_dm_reqs"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	user_settings(user).Render(r.Context(), w)
}