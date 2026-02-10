package routes

import (
	"gochat/db"
	"gochat/pages"
	"gochat/utils"
	"log"
	"net/http"
	"regexp"
	"strconv"
)


func DMchatroomH(w http.ResponseWriter, r *http.Request) {
	id, status, err := utils.GetUserID(r, w)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), id)

	pages.DmChatrooms(user).Render(r.Context(), w)
}


func DMH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r, w)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), userId)

	var id = r.PathValue("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	intId, _ := strconv.ParseInt(id, 10, 64)

	hasUser, err := db.Query.CheckIfUserIsInDM(r.Context(), db.CheckIfUserIsInDMParams{
		ID: intId,
		User1ID: user.ID,
	})

	if err != nil {
		log.Println(utils.GetFuncInfo(), err)
		return
	}

	if !hasUser {
		http.Redirect(w, r, "/dms", http.StatusFound)
		return
	}

	dm, _ := db.Query.GetDMById(r.Context(), intId) 

	pages.DmChat(dm, user).Render(r.Context(), w)
}


func DMactionH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r, w)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), userId)
	
	if !user.Verified {
		http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
		return
	}

	var id = r.URL.Query().Get("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	intId, _ := strconv.ParseInt(id, 10, 64)

	hasUser, err := db.Query.CheckIfUserIsInDM(r.Context(), db.CheckIfUserIsInDMParams{
		ID: intId,
		User1ID: user.ID,
	})

	if err != nil {
		log.Println(utils.GetFuncInfo(), err)
		return
	}

	if !hasUser {
		http.Error(w, "you are not in this dm", http.StatusNotFound)
		return
	}

	switch r.URL.Query().Get("type") {
	case "accept":
		db.Query.UpdateDM(r.Context(), db.UpdateDMParams{
			Status: "accepted",
			ID: intId,
		})

	case "ignore":
		db.Query.UpdateDM(r.Context(), db.UpdateDMParams{
			Status: "ignored",
			ID: intId,
		})

	case "unignore":
		db.Query.UpdateDM(r.Context(), db.UpdateDMParams{
			Status: "accepted",
			ID: intId,
		})

	default:
		http.Error(w, "invalid action", http.StatusBadRequest)
		return
	}
}