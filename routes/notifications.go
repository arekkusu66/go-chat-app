package routes

import (
	"encoding/json"
	"fmt"
	"gochat/models"
	"gochat/utils"
	"net/http"
	"regexp"
)


func NotificationsH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user datas", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("Notifications").First(&user, "id = ?", userData.ID)


	var action = r.URL.Query().Get("action")

	switch action {
		case "get":
			json.NewEncoder(w).Encode(&user.Notifications)
			return
		case "are-there-unread-notifs":
			w.Write([]byte(fmt.Sprint(user.AreThereUnreadNotifs())))
			return
	}


	var id = r.URL.Query().Get("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var notification models.Notification
	models.DB.First(&notification, id)

	if notification.UserID != user.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}


	switch action {
		case "mark-as-read":
			notification.Read = true
			models.DB.Save(&notification)

		case "delete":
			models.DB.Delete(&notification)
	}
}