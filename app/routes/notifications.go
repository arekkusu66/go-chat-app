package routes

import (
	"encoding/json"
	"fmt"
	"gochat/app"
	"net/http"
	"regexp"
	"strconv"
)


func NotificationsH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		var action = r.URL.Query().Get("action")

		switch action {
			case "get":
				notifications, _ := app.Query.GetAllNotifications(r.Context(), userId)
				json.NewEncoder(w).Encode(&notifications)
				return

			case "are-there-unread-notifs":
				areThereUnreadNotifs, _ := app.Query.AreThereUnreadNotifications(r.Context(), userId)
				fmt.Fprint(w, fmt.Sprint(areThereUnreadNotifs))
				return
		}


		var id = r.URL.Query().Get("id")

		if !regexp.MustCompile(`^\d+$`).MatchString(id) {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		intId, _ := strconv.ParseInt(id, 10, 64)

		notification, err := app.Query.GetNoficationById(r.Context(), intId)

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
				if err := app.Query.ReadNotification(r.Context(), intId); err != nil {
					app.Log.Error("couldnt set the notification as read", "error_info", err)
					http.Error(w, "couldnt update the notification", http.StatusInternalServerError)
					return
				}

			case "delete":
				if err := app.Query.DeleteAllNotifications(r.Context(), intId); err != nil {
					app.Log.Error("couldnt delete the notifications", "error_info", err)
					http.Error(w, "couldnt delete the notifications", http.StatusInternalServerError)
					return
				}
		}
	})
}