package routes

import (
	"database/sql"
	"gochat/models"
	"gochat/utils"
	"io"
	"net/http"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func UserPageH(w http.ResponseWriter, r *http.Request) {
	var user, currentUser models.User

	if err := models.DB.Preload(clause.Associations).Preload("DMS.Users").First(&user, "username = ?", r.PathValue("username")).Error; err != nil && err == gorm.ErrRecordNotFound{
		http.Error(w, "user not found!", http.StatusNotFound)
		return
	}

	userData, err := utils.ParseCookie(r)

	if err != nil {
		userpage(user, models.User{}).Render(r.Context(), w)
		return
	}

	models.DB.Preload(clause.Associations).Preload("DMS.Users").First(&currentUser, "id = ?", userData.ID)


	userpage(user, currentUser).Render(r.Context(), w)
}


func EditProfileH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.First(&user, "id = ?", userData.ID)


	switch r.URL.Query().Get("edit") {
		case "description":
			description, err := io.ReadAll(r.Body)

			if err != nil {
				http.Error(w, "error processing the datas", http.StatusInternalServerError)
				return
			}

			w.Write(description)
			
			user.Description = sql.NullString{String: string(description), Valid: true}
			models.DB.Save(&user)
	}
}


func CheckAvailabilityH(w http.ResponseWriter, r *http.Request) {
	var existingUser models.User
	var newUsername = r.PathValue("username")

	if err := models.DB.First(&existingUser, "username = ?", newUsername).Error; err != nil && err == gorm.ErrRecordNotFound {
		w.Write([]byte("username available"))
		return
	}

	if existingUser.ID != "" {
		http.Error(w, "username already in use", http.StatusBadRequest)
		return
	}
}


func EditUsernameH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.First(&user, "id = ?", userData.ID)

	var newUsername = r.URL.Query().Get("username")

	if !utils.Validate("username", newUsername) {
		http.Error(w, "this username is invalid!", http.StatusBadRequest)
		return
	}

	user.Username = newUsername
	models.DB.Save(&user)
}


func UserActionsH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user, targetUser models.User

	var username = r.URL.Query().Get("username")

	if !utils.Validate("username", username) {
		http.Error(w, "username is invalid", http.StatusBadRequest)
		return
	}

	models.DB.Preload(clause.Associations).Preload("DMS.Users").First(&user, "id = ?", userData.ID)

	
	if !user.Verified {
		http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
		return
	}


	if err := models.DB.Preload(clause.Associations).Preload("DMS.Users").First(&targetUser, "username = ?", username).Error; err != nil && err == gorm.ErrRecordNotFound {
		http.Error(w, "this user doesnt exists", http.StatusBadRequest)
		return
	}


	switch r.URL.Query().Get("type") {
		case "add":
			user.Add(&targetUser, w)

		case "cancel":
			user.Cancel(&targetUser, w)

		case "accept":
			user.Accept(&targetUser, w)

		case "ignore":
			user.Ignore(&targetUser, w)

		case "block":
			user.Block(&targetUser, w)

		case "unblock":
			user.Unblock(&targetUser, w)

		case "senddm":
			user.SendDM(&targetUser, w)

		default:
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
	}
}