package models

import (
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)


type ChatDatas struct {
	ChatRoom			ChatRoom
	User				User
	AlreadyJoined		bool
}


type MessageDatas struct {
	IsReply				bool		`json:"isReply"`
	IsBlocked			bool		`json:"isBlocked"`
}


type RateLimiter struct {
	Bucket				chan struct{}
}


type Claims struct {
	ID					string
	jwt.RegisteredClaims
}


type Oauth struct {
	Config				*oauth2.Config
}


type ProviderDatas struct {
	Email				string			`json:"email"`
	Verified			bool			`json:"verified_email"`
}