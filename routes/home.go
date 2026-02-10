package routes

import (
	"gochat/db"
	"gochat/pages"
	"gochat/utils"
	"log"
	"net/http"
)


func HomeH(w http.ResponseWriter, r *http.Request) {
	id, _, err := utils.GetUserID(r, w)

	if err != nil {
		log.Println(utils.GetFuncInfo(), "couldnt get the user datas: ", err)
		pages.Home(nil).Render(r.Context(), w)
		return
	}

	user, err := db.Query.GetUserById(r.Context(), id)
	
	if err != nil {
		log.Println(utils.GetFuncInfo(), "couldnt get the user datas: ", err)
		pages.Home(nil).Render(r.Context(), w)
		return
	}

	pages.Home(&user).Render(r.Context(), w)
}