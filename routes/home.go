package routes

import (
	"gochat/models"
	"gochat/utils"
	"net/http"
)


func HomeH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	userData, err := utils.ParseCookie(r)

	if err != nil {
		home(models.User{}).Render(r.Context(), w)
		return
	}

	var user models.User
	if err := models.DB.Preload("CreatedChats").Preload("JoinedChats").Preload("Notifications").First(&user, "id = ?", userData.ID).Error; err != nil {
		home(models.User{}).Render(r.Context(), w)
		return
	}
	

	home(user).Render(r.Context(), w)
}