package routes

import (
	"encoding/json"
	"fmt"
	"gochat/db"
	"gochat/models"
	"gochat/pages"
	"gochat/types"
	"gochat/utils"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)


func CreateChatH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, err := db.Query.GetUserById(r.Context(), id)

	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if !user.Verified {
		http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
		return
	}

	createdChatsCount, _ := db.Query.GetCreatedChatroomsCount(r.Context(), user.ID)

	if createdChatsCount  >= 30 {
		http.Error(w, "cannot create more than 30 chatrooms", http.StatusBadRequest)
		return
	}

	var title = r.URL.Query().Get("title")

	if strings.TrimSpace(title) == "" {
		title = user.Username + "'s" + " " + "chatroom" + fmt.Sprint(createdChatsCount)
	}

	if err := db.CreateChatroom(r.Context(), user.ID, title); err != nil {
		http.Error(w, "couldnt create the chatroom", http.StatusInternalServerError)
		return
	}
}


func ChatH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r)

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

	chatroomExists, _ := db.Query.CheckChatroomExistence(r.Context(), intId)

	if !chatroomExists {
		http.Error(w, "chatroom not found", http.StatusNotFound)
		return
	}

	chatroom, _ := db.Query.GetChatroom(r.Context(), intId)

	pages.Chat(user, chatroom).Render(r.Context(), w)
}


func ChatActionsH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	var id = r.URL.Query().Get("id")
	
	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	intId, _ := strconv.ParseInt(id, 10, 64)

	alreadyJoined, _ := db.Query.CheckIfAlreadyJoined(r.Context(), db.CheckIfAlreadyJoinedParams{
		UserID: userId,
		ChatroomID: intId,
	})

	switch r.URL.Query().Get("action") {
		case "join":
			if alreadyJoined {
				fmt.Fprint(w, "you havent joined this chatroom!")
				return
			}

			db.Query.JoinChatroom(r.Context(), db.JoinChatroomParams{
				ChatroomID: intId,
				UserID: userId,
			})

			fmt.Fprint(w, (`
				<div id="chat-joined">
					<div id="reply">
						<div id="id-reply">
					</div>
					</div><br /><br />
			
					<input type="text" id="send" placeholder="write a message" style="width:60px;height:35px"/>
					<button onclick="sendMsg()" style="width:60px;height:35px">send</button>
				</div>`))

		case "leave":
			if !alreadyJoined {
				fmt.Fprint(w, "you havent joined this chatroom!")
			}

			db.Query.LeaveChatroom(r.Context(), db.LeaveChatroomParams{
				UserID: userId,
				ChatroomID: intId,
			})

			fmt.Fprintf(w, `
				<h3>join this chat!</h3>
				<button hx-post="/join/chat/%d" hx-trigger="click" hx-target="#join">click to join</button>`, intId)

		default:
			fmt.Fprint(w, "invalid action")
	}
}


func GetOptionsH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	
	var id = r.URL.Query().Get("id")
	
	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	intId, _ := strconv.ParseInt(id, 10, 64)

	message, err := db.Query.GetMessageById(r.Context(), intId)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var cancelBtn = `<button onclick="cancelOptions()">cancel</button>`
	
	if userId == message.UserID {
		fmt.Fprintf(w, "<button onclick=\"deleteMsg(this)\" data-id=\"%s\">delete</button>%s", id, cancelBtn)
		return
	} else {
		fmt.Fprintf(w, "<button onclick=\"reply(this)\" data-id=\"%s\">reply</button>%s", id, cancelBtn)
		return
	}
}


func GetMessageH(w http.ResponseWriter, r *http.Request) {
	userId, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	var id = r.URL.Query().Get("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	intId, _ := strconv.ParseInt(id, 10, 64)

	isReply, _ := db.Query.CheckIfMessageIsReply(r.Context(), db.CheckIfMessageIsReplyParams{
		ID: intId,
		UserID: userId,
	})

	isBlocked, _ := db.Query.CheckMessageUserRelation(r.Context(), db.CheckMessageUserRelationParams{
		ID: intId,
		UserID: userId,
		Type: types.BLOCKED_USER,
	})

	if r.URL.Query().Get("type") == "0" {
		json.NewEncoder(w).Encode(&models.ClientDatas{
			IsReply: isReply,
			IsBlocked: isBlocked,
		})

		return
	}

	datas, err := db.Query.GetFullMessageDatas(r.Context(), intId)

	if err != nil {
		json.NewEncoder(w).Encode(&models.MessageDatas{})
		return
	}

	var messageDatas = models.MessageDatas{
		GetFullMessageDatasRow: datas,
		ClientDatas: models.ClientDatas{
			IsReply: isReply,
			IsBlocked: isBlocked,
		},
	}

	json.NewEncoder(w).Encode(&messageDatas)
}