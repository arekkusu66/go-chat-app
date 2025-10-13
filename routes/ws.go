package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gochat/models"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm/clause"
)


type Hub struct {
	clients			map[*client]struct{}
	rooms			map[string]map[*client]struct{}
	notifs			map[string]map[*client]struct{}
	connect			chan *client
	disconnect		chan *client
	outgoing		chan *outgoing
}


type client struct {
	hub				*Hub
	conn			*websocket.Conn
	outgoing 		chan *outgoing

	Type			string
	id				string
}


type incoming struct {
	Type			string				`json:"type"`
	Data			json.RawMessage		`json:"data"`
}


type outgoing struct {
	Type			string				`json:"type"`
	Data			any					`json:"data"`	
}


var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


func connectClient(client_ *client, rooms_notifs map[string]map[*client]struct{}) {
	if rooms_notifs[client_.id] == nil {
		rooms_notifs[client_.id] = make(map[*client]struct{})
	}

	rooms_notifs[client_.id][client_] = struct{}{}
}


func disconnectClient(client *client, rooms_notifs map[string]map[*client]struct{}) {
	if clients, ok := rooms_notifs[client.id]; ok {
		delete(clients, client)

		if len(clients) == 0 {
			delete(rooms_notifs, client.id)
		}
	}
}


func sendOutgoing(hub *Hub, outgoing *outgoing, rooms_notifs map[string]map[*client]struct{}, id string) {
	if clients, ok := rooms_notifs[id]; ok {
		for client := range clients {
			select {
				case client.outgoing <- outgoing:
				default:
					delete(hub.clients, client)
					close(client.outgoing)
			}
		}
	}
}


func NewHub() *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
		rooms: make(map[string]map[*client]struct{}),
		notifs: make(map[string]map[*client]struct{}),
		connect: make(chan *client),
		disconnect: make(chan *client),
		outgoing: make(chan *outgoing),
	}
}


func (hub *Hub) Run() {
	for {
		select {
			case client_ := <-hub.connect:
				hub.clients[client_] = struct{}{}

				switch client_.Type {
					case types.MSG, types.DEL:
						connectClient(client_, hub.rooms)

					case types.NOTIF:
						connectClient(client_, hub.notifs)
				}

			case client := <-hub.disconnect:
				if _, ok := hub.clients[client]; ok {
					delete(hub.clients, client)
					close(client.outgoing)

					switch client.Type {
						case types.MSG, types.DEL:
							disconnectClient(client, hub.rooms)

						case types.NOTIF:
							disconnectClient(client, hub.notifs)
					}
				}

			case outgoing := <-hub.outgoing:
					switch outgoing.Type {
						case types.MSG, types.DEL:
							var message = outgoing.Data.(*models.Message)
							sendOutgoing(hub, outgoing, hub.rooms, fmt.Sprint(message.ChatRoomID))

						case types.NOTIF:
							var notif = outgoing.Data.(*models.Notification)
							sendOutgoing(hub, outgoing, hub.notifs, notif.UserID)
					}
		}
	}
}


func (c *client) readPump(r *http.Request) {
	defer func() {
		c.hub.disconnect <- c
		c.conn.Close()
		recover()
	}()

	for {
		var incoming incoming
					
		if err := c.conn.ReadJSON(&incoming); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("error reading the message:", err)
			}

			break
		}

		userData, err := utils.ParseCookie(r)

		if err != nil {
			log.Println("could get the user datas: ", err)
			return
		}

		var user models.User
		
		if err := models.DB.Preload(clause.Associations).First(&user, "id = ?", userData.ID).Error; err != nil {
			log.Println("error finding the user: ", err)
			continue
		}

		switch incoming.Type {
			case types.MSG:
				var message = &models.Message{
					Date: time.Now(),
					User: user,
					UserID: user.ID,
				}

				var reply models.Message

				if err := json.Unmarshal(incoming.Data, message); err != nil {
					log.Println("invalid message: ", err)
					continue
				}

				if message.ReplyID != 0 {
					if models.DB.First(&reply, message.ReplyID).Error == nil {
						message.Reply = &reply
					}
				}

				if message.ChatRoomID == 0 {
					var chatroom models.ChatRoom
					if err := models.DB.First(&chatroom, message.ChatRoomID).Error; err != nil {
						log.Println("error finding the message: ", err)
						continue
					}
					message.ChatOp = sql.NullString{String: chatroom.CreatedBy, Valid: true}
				}

				models.DB.Create(&message)

				c.hub.outgoing <- &outgoing{Type: types.MSG, Data: message}


			case types.DEL:
				var message models.Message

				if err := json.Unmarshal(incoming.Data, &message); err != nil {
					log.Println("invalid message: ", err)
					continue
				}

				id, roomID := message.ID, message.ChatRoomID

				if err := models.DB.Delete(&message).Error; err != nil {
					log.Println(err)
					continue
				}
				
				c.hub.outgoing <- &outgoing{Type: types.DEL, Data: &models.Message{ID: id, ChatRoomID: roomID}}


			case types.NOTIF:
				var targetUser models.User

				var notif = &models.Notification{
					Date: time.Now(),
				}

				if err := json.Unmarshal(incoming.Data, notif); err != nil {
					log.Println("invalid notification: ", err)
					continue
				}

				if err := models.DB.Preload(clause.Associations).First(&targetUser, "username = ?", notif.User.Username).Error; err != nil {
					log.Println(err)
					continue
				}

				if targetUser.CheckUserRelations(user, targetUser.BlockedUsers) {
					continue
				}

				notif.User = targetUser
				notif.NotifFrom = user.ID

			
				switch notif.Type {
					case types.FRIEND_REQ:
						if targetUser.CheckUserRelations(targetUser, user.SentFriendReqs) {
							continue
						}

						notif.Message = user.Username + " sent you a friend request"
						notif.Link = "/user/" + user.Username

					case types.DM_REQ:
						if targetUser.CheckUserRelations(targetUser, user.DMedUsers) {
							continue
						}

						notif.Message = user.Username + " sent you a message request"
						notif.Link = fmt.Sprintf("/dm/%d", user.GetDMid(targetUser))

					case types.ACCEPT_FRIEND_REQ:
						if targetUser.CheckUserRelations(targetUser, user.Friends) {
							continue
						}

						notif.Message = user.Username + " accepted your friend request"
						notif.Link = "/user/" + user.Username

					default:
						continue
				}

				models.DB.Create(notif)

				c.hub.outgoing <- &outgoing{Type: types.NOTIF, Data: notif}
		}
	}
}


func (c *client) writePump() {
	defer func() {
		c.conn.Close()
		recover()
	}()

	for {
		select {
			case outgoing, ok := <-c.outgoing:
				if !ok {
					c.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}

				c.conn.WriteJSON(outgoing)

			case <-time.Tick(time.Minute):
		}
	}
}


func (hub *Hub) WSHandler(Type string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Fatal("error trying to connect to the websocket: ", err)
		}

		var client = &client{
			hub: hub,
			conn: conn,
			outgoing: make(chan *outgoing, 256),
			Type: Type,
		}


		switch Type {
			case types.MSG, types.DEL:
				client.id = r.PathValue("id")
			case types.NOTIF:
				userData, err := utils.ParseCookie(r)

				if err != nil {
					log.Fatal("error getting the user datas: ", err)
				}

				client.id = userData.ID
		}

		go client.readPump(r)
		go client.writePump()

		client.hub.connect <- client
	})
}