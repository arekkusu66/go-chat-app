package routes

import (
	"database/sql"
	"fmt"
	"gochat/models"
	"gochat/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm/clause"
)


func MSGWS(w http.ResponseWriter, r *http.Request, clients map[*websocket.Conn]string, isItChatRoom bool) {
	conn, err := utils.Wupg.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	var mu sync.Mutex

	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()

		conn.Close()
	}()

	var id = r.PathValue("id")

	mu.Lock()
	clients[conn] = id
	mu.Unlock()


	userData, err := utils.ParseCookie(r)

	if err != nil {
		fmt.Println(err)
		return
	}

	var user models.User
	models.DB.First(&user, "id = ?", userData.ID)

	var reply models.Message

	
	for {
		var message = models.Message{
			Date: time.Now(),
			User: user,
			UserID: user.ID,
		}

		if err := conn.ReadJSON(&message); err != nil {
			fmt.Println(err)
			return
		}


		if message.ReplyID != 0 {
			if models.DB.First(&reply, message.ReplyID).Error == nil {
				message.Reply = &reply
			}
		}


		if isItChatRoom {
			var chatroom models.ChatRoom
			models.DB.First(&chatroom, message.ChatRoomID)
			message.ChatOp = sql.NullString{String: chatroom.CreatedBy, Valid: true}
		}

		models.DB.Create(&message)


		go func() {
			for client, wsid := range clients {
				if wsid == id {
					client.WriteJSON(&message)
				}
			}
		}()
	}
}


func DELWS(w http.ResponseWriter, r *http.Request, clients map[*websocket.Conn]string) {
	conn, err := utils.Wupg.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	var mu sync.Mutex

	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()

		conn.Close()
	}()

	mu.Lock()
	clients[conn] = ""
	mu.Unlock()


	for {
		mtype, msg, err := conn.ReadMessage()

		if err != nil {
			fmt.Println(err)
			return
		}

		var id = string(msg)

		models.DB.Delete(&models.Message{}, id)
		models.DB.Model(&models.Message{}).Where("reply_id = ?", id).Updates(models.Message{ReplyStatus: "deleted"})


		go func() {
			for client := range clients {
				client.WriteMessage(mtype, msg)
			}
		}()
	}
}


func NOTIFWS(w http.ResponseWriter, r *http.Request, clients map[*websocket.Conn]string) {
	conn, err := utils.Wupg.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	var mu sync.Mutex

	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()
		
		conn.Close()
	}()


	userData, err := utils.ParseCookie(r)

	if err != nil {
		fmt.Println(err)
		return
	}

	var user models.User
	models.DB.Preload(clause.Associations).First(&user, "id = ?", userData.ID)

	mu.Lock()
	clients[conn] = user.ID
	mu.Unlock()
	
	var targetUser models.User


	for {
		var notification = models.Notification{
			Date: time.Now(),
		}

		if err := conn.ReadJSON(&notification); err != nil {
			fmt.Println(err)
			return
		}

		if err := models.DB.Preload(clause.Associations).First(&targetUser, "username = ?", notification.User.Username).Error; err != nil {
			fmt.Println(err)
			return
		}


		if targetUser.CheckUserRelations(user, targetUser.BlockedUsers) {
			return
		}


		notification.User = targetUser
		notification.NotifFrom = user.ID


		switch notification.Type {
			case "friend-req":
				if targetUser.CheckUserRelations(targetUser, user.SentFriendReqs) {
					return
				}

				notification.Message = user.Username + " sent you a friend request"
				notification.Link = "/user/" + user.Username

			case "dm-req":
				if targetUser.CheckUserRelations(targetUser, user.DMedUsers) {
					return
				}

				notification.Message = user.Username + " sent you a message request"
				notification.Link = fmt.Sprintf("/dm/%d", user.GetDMid(targetUser))

			case "accept-friend-req":
				if targetUser.CheckUserRelations(targetUser, user.Friends) {
					return
				}

				notification.Message = user.Username + " accepted your friend request"
				notification.Link = "/user/" + user.Username

			default:
				return
		}

		models.DB.Create(&notification)

		
		go func() {
			for client, id := range clients {
				if id == targetUser.ID {
					client.WriteMessage(websocket.TextMessage, []byte(fmt.Sprint(targetUser.UnreadNotifsCount())))
				}
			}
		}()
	}
}