package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"gochat/app"
	"gochat/app/db"
	"gochat/app/models"
	"gochat/app/types"

	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)


var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


type client struct {
	conn      *websocket.Conn
	nc	      *nats.Conn

	outgoing  chan *outgoing

	kind	  types.WSMessage
	id		  string
}


type incoming struct {
	Kind			types.WSMessage		`json:"kind"`
	Data			json.RawMessage		`json:"data"`
}


type outgoing struct {
	Kind			types.WSMessage		`json:"kind"`
	Data			any					`json:"data"`	
}


func (c *client) Sub() (*nats.Subscription, error) {
	var subj string

	switch c.kind {
	case types.MSG, types.DEL:
		subj = "room."
	case types.NOTIF:
		subj = "notif."
	}

	return c.nc.Subscribe(
		subj + c.id,
		func(msg *nats.Msg) {
			var outgoing = &outgoing{}

			if err := json.Unmarshal(msg.Data, outgoing); err != nil {
				return
			}

			select {
			case c.outgoing <- outgoing:
			default:
			}
		},
	)
}


func (c *client) Pub(outgoing *outgoing) error {
	data, err := json.Marshal(outgoing)

	if err != nil {
		return err
	}

	var subj string

	switch c.kind {
	case types.MSG, types.DEL:
		subj = "room."
	case types.NOTIF:
		subj = "notif."
	}

	return c.nc.Publish(subj + c.id, data)
}


func (c *client) readPump(app *app.App, sub *nats.Subscription,  r *http.Request, w http.ResponseWriter) {
	defer func() {
		c.conn.Close()
		close(c.outgoing)
		sub.Drain()
	}()

	for {
		var incoming = &incoming{}
					
		if err := c.conn.ReadJSON(incoming); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				app.Log.Error("error reading the message", "error_info", err)
			}

			break
		}

		userId, _, err := app.GetUserID(r, w)

		if err != nil {
			app.Log.Error("couldnt get the user datas", "error_info", err)
			continue
		}

		user, err := app.Query.GetUserById(context.Background(), userId)

		if err != nil {
			app.Log.Error("couldnt find the user", "error_info", err)
			continue
		}

		switch incoming.Kind {
			case types.MSG:				
				var message  = &db.CreateMessageParams{
					UserID: user.ID,
				}

				if err := json.Unmarshal(incoming.Data, &message); err != nil {
					app.Log.Error("couldnt unmarshal the message", "error_info", err)
					continue
				}

				id, err := app.Query.CreateMessage(context.Background(), *message)

				if err != nil {
					app.Log.Error("couldnt create the message", "error_info", err)
					continue
				}

				messageDatas, err := app.Query.GetFullMessageDatas(context.Background(), id)

				if err != nil {
					app.Log.Error("couldnt get the message datas", "error_info", err)
					continue
				}

				if err := c.Pub(&outgoing{Kind: types.MSG, Data: &models.MessageDatas{
					GetFullMessageDatasRow: messageDatas,
				}}); err != nil {
					app.Log.Error("couldnt broadcast the message to the user", "error_info", err)
					continue
				}


			case types.DEL:
				var message db.Message

				if err := json.Unmarshal(incoming.Data, &message); err != nil {
					app.Log.Error("coudlnt unmarshal the message", "error_info", err)
					continue
				}

				id, chatId, dmId := message.ID, message.ChatroomID, message.DmID

				if err := db.DeleteMessage(context.Background(), app.DB, app.Query, id); err != nil {
					app.Log.Error("couldnt delete the message", "error_info", err)
					continue
				}

				if err := c.Pub(&outgoing{
					Kind: types.DEL, 
					Data: &db.Message{ID: id, ChatroomID: chatId, DmID: dmId},
				}); err != nil {
					app.Log.Error("couldnt send the ougoing to the user", "error_info", err)
					continue
				}


			case types.NOTIF:
				var notifParams = &db.CreateNotificationParams{}

				if err := json.Unmarshal(incoming.Data, &notifParams); err != nil {
					app.Log.Error("couldnt unmarshal the notification", "error_info", err)
					continue
				}

				notifParams.NotifFrom = user.ID.String()
			
				switch notifParams.Kind {
					case "friend_req":
						notifParams.Message = user.Username + " sent you a friend request"
						notifParams.Link = "/user/" + user.Username

					case "dm_req":
						dm, _ := app.Query.GetDMWithBothUsers(context.Background(), db.GetDMWithBothUsersParams{
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

				if err := app.Query.CreateNotification(context.Background(), *notifParams); err != nil {
					continue
				}

				if err := c.Pub(&outgoing{Kind: types.NOTIF, Data: notifParams}); err != nil {
					app.Log.Error("couldnt send the ougoing to the user", "error_info", err)
					continue
				}
		}
	}
}


func (c *client) writePump(app *app.App) {
	defer func() {
		c.conn.Close()
	}()

	for outgoing := range c.outgoing {
		if err := c.conn.WriteJSON(outgoing); err != nil {
			app.Log.Error("couldnt broadcast the message to the user", "error_info", err)
		}
	}
}


func WSHandler(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			app.Log.Error("error trying to connect to the websocket", "error_info",err)
			return
		}

		var kind = r.PathValue("kind")

		var client = &client{
			conn: conn,
			nc: app.NcConn,
			outgoing: make(chan *outgoing, 256),
			kind: types.WSMessage(kind),
		}

		switch kind {
			case string(types.MSG), string(types.DEL):
				client.id = r.PathValue("id")
			case string(types.NOTIF):
				id, _, err := app.GetUserID(r, w)

				if err != nil {
					app.Log.Error("couldnt connect the user to the notif ws", "error_info", err)
					return
				}

				client.id = id.String()
		}

		sub, err := client.Sub()

		if err != nil {
			app.Log.Error("couldnt subscribe the user to the nats connection", "error_info", err)
			return
		}

		go client.readPump(app, sub, r, w)
		go client.writePump(app)
	})
}