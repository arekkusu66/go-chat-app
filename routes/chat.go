package routes

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"gochat/models"
	"gochat/utils"
	"net/http"
	"regexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


func CreateChatH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var chatroomData models.ChatRoom
	var user models.User

	userData, err := utils.ParseCookie(r)

	if err != nil {
		w.Write([]byte("couldnt retrieve user data"))
		return
	}

	models.DB.Preload("CreatedChats").First(&user, "id = ?", userData.ID)

	if !user.Verified {
		http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
		return
	}


	var reqBuf = new(bytes.Buffer)

	if _, err := bufio.NewReader(r.Body).WriteTo(reqBuf); err != nil {
		fmt.Println(err)
		return
	}


	json.Unmarshal(reqBuf.Bytes(), &chatroomData)

	if chatroomData.Title == "" {
		http.Error(w, "the name of the chatroom cannot be empty!", http.StatusBadRequest)
		return
	}

	var newChatroom = models.ChatRoom{
		Title: chatroomData.Title,
		CreatedBy: user.ID,
		CreatedByName: user.Username,
	}

	user.CreatedChats = append(user.CreatedChats, newChatroom)
	models.DB.Save(&user)

	if len(user.CreatedChats) > 30 {
		http.Error(w, "you cant create more than 30 chatrooms!", http.StatusBadRequest)
		return
	}
}


func ChatH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var chatroom models.ChatRoom
	var user models.User

	models.DB.Preload("CreatedChats").Preload("JoinedChats").Preload("Messages").Preload("BlockedUsers").First(&user, "id = ?", userData.ID)
	

	var id = r.PathValue("id")

	if !regexp.MustCompile(`^\d+$`).MatchString(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := models.DB.Preload("Messages.User").Preload("Messages.User.BlockedUsers").Preload("Messages.Reply").Preload(clause.Associations).First(&chatroom, id).Error; err != nil && err == gorm.ErrRecordNotFound {
		http.Error(w, "chatroom not found", http.StatusBadRequest)
		return
	}


	var chatroomDatas = models.ChatDatas{
		ChatRoom: chatroom,
		User: user,
		AlreadyJoined: user.AlreadyJoined(chatroom),
	}
	

	chat(chatroomDatas).Render(r.Context(), w)
}


func JoinChatH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		w.Write([]byte("you are not logged in!"))
		return
	}

	var user models.User
	models.DB.Preload("CreatedChats").Preload("JoinedChats").First(&user, "id = ?", userData.ID)

	if !user.Verified {
		http.Error(w, "you need to be verified in order to do that", http.StatusForbidden)
		return
	}

	var chatroom models.ChatRoom
	models.DB.First(&chatroom, r.PathValue("id"))

	if user.AlreadyJoined(chatroom) {
		w.Write([]byte("you already joined this chat!"))
		return
	}

	user.JoinedChats = append(user.JoinedChats, chatroom)
	models.DB.Save(&user)

	w.Write([]byte(`<div id="chat-joined"><div id="reply"><div id="id-reply"></div></div><br /><br /><input type="text" id="send" placeholder="write a message" style="width:60px;height:35px"/><button onclick="sendMsg()" style="width:60px;height:35px">send</button></div>`))
}


func LeaveChatH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("JoinedChats").First(&user, "id = ?", userData.ID)

	var chatroom models.ChatRoom
	models.DB.First(&chatroom, r.PathValue("id"))

	models.DB.Model(&user).Association("JoinedChats").Delete(&chatroom)

	w.Write([]byte(fmt.Sprintf("<h3>join this chat!</h3><button hx-post=\"/join/chat/%d\" hx-trigger=\"click\" hx-target=\"#join\">click to join</button>", chatroom.ID)))
}


func GetOptionsH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}
	
	var user models.User
	models.DB.First(&user, "id = ?", userData.ID)

	
	var id = r.URL.Query().Get("id")

	var message models.Message
	if err := models.DB.First(&message, id).Error; err == gorm.ErrRecordNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	}


	var cancelBtn = `<button onclick="cancelOptions()">cancel</button>`
	
	if user.ID == message.UserID {
		w.Write([]byte(fmt.Sprintf("<button onclick=\"deleteMsg(this)\" data-id=\"%s\">delete</button>%s", id, cancelBtn)))
		return
	} else {
		w.Write([]byte(fmt.Sprintf("<button onclick=\"reply(this)\" data-id=\"%s\">reply</button>%s", id, cancelBtn)))
		return
	}
}


func GetMessageH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("Messages").Preload("BlockedUsers").First(&user, "id = ?", userData.ID)

	var id = r.URL.Query().Get("id")

	var message models.Message
	if err := models.DB.Preload(clause.Associations).Preload("User.BlockedUsers").First(&message, id).Error; err != nil && err == gorm.ErrRecordNotFound {
		http.Error(w, "message not found", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(&message)
}


func GetDatasH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("Messages").Preload("BlockedUsers").First(&user, "id = ?", userData.ID)


	var id = r.URL.Query().Get("id")

	var message models.Message
	if err := models.DB.Preload("User.BlockedUsers").First(&message, id).Error; err != nil && err == gorm.ErrRecordNotFound {
		json.NewEncoder(w).Encode(&models.MessageDatas{IsReply: false, IsBlocked: false})
		return
	}

	var messageDatas = models.MessageDatas{
		IsReply: utils.IsReply(user, fmt.Sprint(message.ReplyID)),
		IsBlocked: user.CheckUserRelations(message.User, user.BlockedUsers),
	}

	json.NewEncoder(w).Encode(&messageDatas)
}