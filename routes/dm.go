package routes

import (
	"gochat/models"
	"gochat/utils"
	"net/http"
	"regexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func DMchatroomH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload(clause.Associations).Preload("DMS.Users").Preload("DMRequests.Users").Preload("IgnoredDMS.Users").Preload("DMS.RequestToUser").First(&user, "id = ?", userData.ID)

	dmchatrooms(user).Render(r.Context(), w)
}


func DMH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload(clause.Associations).Preload("DMS.Users").First(&user, "id = ?", userData.ID)


	var dm models.DM
	var id = r.PathValue("id")

	if err := models.DB.Preload(clause.Associations).Preload("Messages.User").First(&dm, id).Error; err != nil && err == gorm.ErrRecordNotFound {
		http.Redirect(w, r, "/dms", http.StatusFound)
		return
	}


	if !dm.HasUser(user) {
		http.Redirect(w, r, "/dms", http.StatusFound)
		return
	}


	dmchat(dm, user).Render(r.Context(), w)
}


func DMactionH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user datas", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload(clause.Associations).First(&user, "id = ?", userData.ID)

	
	if !user.Verified {
		http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
		return
	}


	var id = r.URL.Query().Get("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var dm models.DM
	if err := models.DB.Preload(clause.Associations).First(&dm, id).Error; err != nil && err == gorm.ErrRecordNotFound {
		http.Error(w, "dm chat not found", http.StatusNotFound)
		return
	}

	if !dm.HasUser(user) {
		return
	}


	switch r.URL.Query().Get("type") {
		case "accept":
			user.AcceptDM(&dm, w)
			
		case "reject":
			user.IgnoreDM(&dm)

		default:
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
	}
}