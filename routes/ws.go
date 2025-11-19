package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"gochat/db"
	"gochat/models"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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

	Type			types.WSMessage
	id				string
}


type incoming struct {
	Type			types.WSMessage		`json:"type"`
	Data			json.RawMessage		`json:"data"`
}


type outgoing struct {
	Type			types.WSMessage		`json:"type"`
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
					case types.MSG:
						var message = outgoing.Data.(*models.MessageDatas)

						if message.ChatroomID.Valid {
							sendOutgoing(hub, outgoing, hub.rooms, 
								fmt.Sprint(message.GetFullMessageDatasRow.ChatroomID.Int64))
						} else {
							sendOutgoing(hub, outgoing, hub.rooms, 
								fmt.Sprint(message.GetFullMessageDatasRow.DmID.Int64))
						}

					case types.DEL:
						var message = outgoing.Data.(*db.Message)

						if message.ChatroomID.Valid {
							sendOutgoing(hub, outgoing, hub.rooms, 
								fmt.Sprint(message.ChatroomID.Int64))
						} else {
							sendOutgoing(hub, outgoing, hub.rooms, 
								fmt.Sprint(message.DmID.Int64))
						}

					case types.NOTIF:
						var notif = outgoing.Data.(*db.CreateNotificationParams)
						sendOutgoing(hub, outgoing, hub.notifs, notif.UserID.String())
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

		userId, _, err := utils.GetUserID(r)

		if err != nil {
			log.Println(utils.GetFuncInfo(), err)
			continue
		}

		user, err := db.Query.GetUserById(context.Background(), userId)

		if err != nil {
			log.Println(utils.GetFuncInfo(), err)
			continue
		}

		switch incoming.Type {
			case types.MSG:
				// fmt.Println("incoming msg", string(incoming.Data))
				
				var message  = &db.CreateMessageParams{
					UserID: user.ID,
				}

				if err := json.Unmarshal(incoming.Data, &message); err != nil {
					log.Println("invalid message: ", err)
					continue
				}

				id, err := db.Query.CreateMessage(context.Background(), *message)

				if err != nil {
					log.Println(utils.GetFuncInfo(), err)
					continue
				}

				messageDatas, err := db.Query.GetFullMessageDatas(context.Background(), id)

				if err != nil {
					log.Println(utils.GetFuncInfo(), err)
					continue
				}

				c.hub.outgoing <- &outgoing{Type: types.MSG, Data: &models.MessageDatas{
					GetFullMessageDatasRow: messageDatas,
				}}


			case types.DEL:
				var message db.Message

				if err := json.Unmarshal(incoming.Data, &message); err != nil {
					log.Println("invalid message: ", err)
					continue
				}

				id, chatId, dmId := message.ID, message.ChatroomID, message.DmID
				
				c.hub.outgoing <- &outgoing{
					Type: types.DEL, 
					Data: &db.Message{ID: id, ChatroomID: chatId, DmID: dmId},
				}

				if err := db.DeleteMessage(context.Background(), id); err != nil {
					log.Println("couldnt delete the message", err)
					continue
				}


			case types.NOTIF:
				// fmt.Println("incoming data notif:", string(incoming.Data))

				var notifParams = &db.CreateNotificationParams{}

				if err := json.Unmarshal(incoming.Data, &notifParams); err != nil {
					log.Println("invalid notification: ", err)
					continue
				}

				notifParams.NotifFrom = user.ID.String()
			
				switch notifParams.Type {
					case "friend_req":
						notifParams.Message = user.Username + " sent you a friend request"
						notifParams.Link = "/user/" + user.Username

					case "dm_req":
						dm, _ := db.Query.GetDMWithBothUsers(context.Background(), db.GetDMWithBothUsersParams{
							User1ID: user.ID,
							User2ID: notifParams.UserID,
						})

						notifParams.Message = user.Username + " sent you a message request"
						notifParams.Link = "/dm/" + fmt.Sprint(dm.ID)

					case "accepted_friend_req":
						notifParams.Message = user.Username + " accepted your friend request"
						notifParams.Link = "/user/" + user.Username

					default:
						continue
				}

				if err := db.Query.CreateNotification(context.Background(), *notifParams); err != nil {
					log.Println(utils.GetFuncInfo(), err)
					continue
				}

				c.hub.outgoing <- &outgoing{Type: types.NOTIF, Data: notifParams}
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

				// fmt.Printf("%#v\n", outgoing.Data)

				if err := c.conn.WriteJSON(outgoing); err != nil {
					log.Println(utils.GetFuncInfo(), err)
				}

			case <-time.Tick(time.Minute):
		}
	}
}


func (hub *Hub) WSHandler(Type types.WSMessage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Println("error trying to connect to the websocket: ", err)
			return
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
				id, _, err := utils.GetUserID(r)

				if err != nil {
					log.Println("couldnt connect the user to the notif ws", err)
					return
				}

				client.id = id.String()
		}

		go client.readPump(r)
		go client.writePump()

		client.hub.connect <- client
	})
}