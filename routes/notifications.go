package routes

import (
	"encoding/json"
	"fmt"
	"gochat/db"
	"gochat/utils"
	"log"
	"net/http"
	"regexp"
	"strconv"
)


func NotificationsH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r, w)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	var action = r.URL.Query().Get("action")

	switch action {
		case "get":
			notifications, err := db.Query.GetAllNotifications(r.Context(), userId)

			if err == nil {
				json.NewEncoder(w).Encode(&notifications)
				return
			} else {
				log.Println(utils.GetFuncInfo(), err)
				json.NewEncoder(w).Encode(&[]db.Notification{})
				return
			}

		case "are-there-unread-notifs":
			areThereUnreadNotifs, _ := db.Query.AreThereUnreadNotifications(r.Context(), userId)
			fmt.Fprint(w, fmt.Sprint(areThereUnreadNotifs))
			return
	}


	var id = r.URL.Query().Get("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	intId, _ := strconv.ParseInt(id, 10, 64)

	notification, err := db.Query.GetNoficationById(r.Context(), intId)

	if err != nil {
		http.Error(w, "notification not found", http.StatusNotFound)
		return
	}

	if notification.UserID != userId {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	
	switch action {
		case "mark-as-read":
			if err := db.Query.ReadNotification(r.Context(), intId); err != nil {
				log.Println(utils.GetFuncInfo(), err)
				http.Error(w, "couldnt update the notification", http.StatusInternalServerError)
				return
			}

		case "delete":
			if err := db.Query.DeleteAllNotifications(r.Context(), intId); err != nil {
				log.Println(utils.GetFuncInfo(), err)
				http.Error(w, "couldnt delete the notification", http.StatusInternalServerError)
				return
			}
	}
}