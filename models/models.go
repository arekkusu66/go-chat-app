package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)


type ChatDatas struct {
	ChatRoom			ChatRoom
	User				User
	AlreadyJoined		bool
}


type Client struct {
	LastRequest			time.Time
	Limiter				*rate.Limiter
}


type Claims struct {
	ID					string
	jwt.RegisteredClaims
}


type WSconf struct {
	Clients				map[*websocket.Conn]string
	Type				string
}


type Oauth struct {
	Config				*oauth2.Config
}


type ProviderDatas struct {
	Email				string			`json:"email"`
	Verified			bool			`json:"verified_email"`
}