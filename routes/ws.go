package routes

import (
	"database/sql"
	"fmt"
	"gochat/models"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm/clause"
)


type WebsocketServer struct {
	upgrader			websocket.Upgrader
	
	mu					sync.RWMutex

	connect				chan *connect
	disconnect			chan *disconnect
	broadcast			chan *broadcast

	clientsMsg			map[*websocket.Conn]string
	clientsNotif		map[*websocket.Conn]string
}


type connect struct {
	conn				*websocket.Conn
	type_				string
	id					string
}


type disconnect struct {
	conn				*websocket.Conn
	type_				string
}


type broadcast struct {
	data 				any
	type_				string
	id					string
}


func NewServer() *WebsocketServer {
	return &WebsocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},

		connect: make(chan *connect),
		disconnect: make(chan *disconnect),
		broadcast: make(chan *broadcast),

		clientsMsg: make(map[*websocket.Conn]string),
		clientsNotif: make(map[*websocket.Conn]string),

		mu: sync.RWMutex{},
	}
}


func (ws *WebsocketServer) Run() {
	for {
		select {
			case conn := <-ws.connect:
				ws.mu.Lock()
				switch conn.type_ {
					case types.MSG:
						ws.clientsMsg[conn.conn] = conn.id
					case types.NOTIF:
						ws.clientsNotif[conn.conn] = conn.id
				}
				ws.mu.Unlock()

			case diss := <-ws.disconnect:
				ws.mu.Lock()
				switch diss.type_ {
					case types.MSG:
						delete(ws.clientsMsg, diss.conn)
						diss.conn.Close()
					case types.NOTIF:
						delete(ws.clientsNotif, diss.conn)
						diss.conn.Close()
				}
				ws.mu.Unlock()

			case broadcast := <-ws.broadcast:
				ws.mu.RLock()

				switch broadcast.type_ {
					case types.MSG:
						for client, roomID := range ws.clientsMsg {
							if roomID == broadcast.id {
								if err := client.WriteJSON(broadcast.data); err != nil {
									log.Println(err)
								}
							}
						}

					case types.NOTIF:
						for client, userID := range ws.clientsNotif {
							if userID == broadcast.id {
								if err := client.WriteJSON(broadcast.data); err != nil {
									log.Println(err)
								}
						}
					}
				}

				ws.mu.RUnlock()
		}
	}
}


func (ws *WebsocketServer) MSGWS(isItChatRoom bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := ws.upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Fatal(err)
		}

		defer func() {
			ws.disconnect <- &disconnect{
				conn: conn,
				type_: types.MSG,
			}
		}()

		var roomID = r.PathValue("id")

		ws.connect <- &connect{
			conn: conn,
			id: roomID,
			type_: types.MSG,
		}


		userData, err := utils.ParseCookie(r)

		if err != nil {
			return
		}

		var user models.User
		models.DB.First(&user, "id = ?", userData.ID)

		var reply models.Message

	
		for {
			var message = &models.Message{
				Date: time.Now(),
				User: user,
				UserID: user.ID,
			}

			if err := conn.ReadJSON(&message); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {}
				break 
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

			ws.broadcast <- &broadcast{
				data: message,
				id: roomID,
				type_: types.MSG,
			}
		}
	})
}


func (ws *WebsocketServer) DELWS(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		ws.disconnect <- &disconnect{
			conn: conn,
			type_: types.MSG,
		}
	}()

	var roomID = r.PathValue("id")

	ws.connect <- &connect{
		conn: conn,
		id: roomID,
		type_: types.MSG,
	}

	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {}
			break 
		}

		var id = string(msg)

		models.DB.Delete(&models.Message{}, id)
		models.DB.Model(&models.Message{}).Where("reply_id = ?", id).Updates(models.Message{ReplyStatus: "deleted"})

		ws.broadcast <- &broadcast{
			data: map[string]string{"id": id},
			id: roomID,
			type_: types.MSG,
		}
	}
}


func (ws *WebsocketServer) NOTIFWS(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)

	if err != nil {
		return
	}

	defer func ()  {
		ws.disconnect <- &disconnect{
			conn: conn,
			type_: types.NOTIF,
		}
	}()

	userData, err := utils.ParseCookie(r)

	if err != nil {
		return
	}

	var user models.User
	models.DB.Preload(clause.Associations).First(&user, "id = ?", userData.ID)


	ws.connect <- &connect{
		conn: conn,
		id: user.ID,
		type_: types.NOTIF,
	}
	

	var targetUser models.User


	for {
		var notification = &models.Notification{
			Date: time.Now(),
		}

		if err := conn.ReadJSON(notification); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {}
			break 
		}

		if err := models.DB.Preload(clause.Associations).First(&targetUser, "username = ?", notification.User.Username).Error; err != nil {
			return
		}


		if targetUser.CheckUserRelations(user, targetUser.BlockedUsers) {
			return
		}


		notification.User = targetUser
		notification.NotifFrom = user.ID


		switch notification.Type {
			case types.FRIEND_REQ:
				if targetUser.CheckUserRelations(targetUser, user.SentFriendReqs) {
					return
				}

				notification.Message = user.Username + " sent you a friend request"
				notification.Link = "/user/" + user.Username

			case types.DM_REQ:
				if targetUser.CheckUserRelations(targetUser, user.DMedUsers) {
					return
				}

				notification.Message = user.Username + " sent you a message request"
				notification.Link = fmt.Sprintf("/dm/%d", user.GetDMid(targetUser))

			case types.ACCEPT_FRIEND_REQ:
				if targetUser.CheckUserRelations(targetUser, user.Friends) {
					return
				}

				notification.Message = user.Username + " accepted your friend request"
				notification.Link = "/user/" + user.Username

			default:
				return
		}

		models.DB.Create(notification)
		
		ws.broadcast <- &broadcast{
			data: map[string]int64{"count": targetUser.UnreadNotifsCount()},
			id: targetUser.ID,
			type_: types.NOTIF,
		}
	}
}