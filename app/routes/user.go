package routes

import (
	"gochat/app"
	"gochat/app/db"
	"gochat/app/pages"
	"gochat/app/utils"
	"io"
	"net/http"
)


func UserPageH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := app.Query.GetUserByName(r.Context(), r.PathValue("username"))

		if err != nil {
			pages.UserPage(app, db.User{}, db.User{}).Render(r.Context(), w)
			return
		}

		id, _, err := app.GetUserID(r, w)

		if err != nil {
			pages.UserPage(app, user, db.User{}).Render(r.Context(), w)
			return
		}

		currentUser, _ := app.Query.GetUserById(r.Context(), id)

		pages.UserPage(app, user, currentUser).Render(r.Context(), w)
	})
}


func EditProfileH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		switch r.URL.Query().Get("edit") {
			case "description":
				description, err := io.ReadAll(r.Body)

				if err != nil {
					http.Error(w, "error processing the datas", http.StatusInternalServerError)
					return
				}

				if err := app.Query.UpdateUser(r.Context(), db.UpdateUserParams{
					ID: id,
					Description: string(description),
				}); err != nil {
					http.Error(w, "error saving the datas", http.StatusInternalServerError)
					return
				}

				w.Write(description)
		}
	})
}


func EditUsernameH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		var newUsername = r.URL.Query().Get("username")

		usernameExists, _ := app.Query.CheckUsernameAvailability(r.Context(), newUsername)

		if usernameExists {
			http.Error(w, "this is username is already in use", http.StatusBadRequest)
			return
		}

		if !utils.Validate("username", newUsername) {
			http.Error(w, "this username is invalid!", http.StatusBadRequest)
			return
		}

		if err := app.Query.UpdateUser(r.Context(), db.UpdateUserParams{
			ID: id,
			Username: newUsername,
		}); err != nil {
			http.Error(w, "couldnt update the username", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}


func UserActionsH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		user, _ := app.Query.GetUserById(r.Context(), id)
		
		if !user.Verified {
			http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
			return
		}

		var username = r.URL.Query().Get("username")

		if !utils.Validate("username", username) {
			http.Error(w, "username is invalid", http.StatusBadRequest)
			return
		}

		targetUser, err := app.Query.GetUserByName(r.Context(), username)
		
		if err != nil {
			http.Error(w, "this user doesnt exists", http.StatusBadRequest)
			return
		}

		switch r.URL.Query().Get("type") {
			case "add":
				if status, err := db.AddFriend(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			case "cancel":
				if status, err := db.CancelFriendRequest(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			case "accept":
				if status, err := db.AcceptFriendRequest(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			case "ignore":
				if status, err := db.IgnoreFriendRequest(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			case "block":
				if status, err := db.BlockUser(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			case "unblock":
				if status, err := db.UnblockUser(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			case "senddm":
				if status, err := db.SendDM(r.Context(), app.DB, app.Query, user.ID, targetUser.ID); err != nil {
					http.Error(w, err.Error(), status)
					return
				}

			default:
				http.Error(w, "invalid action", http.StatusBadRequest)
				return
		}
	})
}