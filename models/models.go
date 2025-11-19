package models

import (
	"gochat/db"
)

type ClientDatas struct {
	IsReply				bool		`json:"is_reply"`
	IsBlocked			bool		`json:"is_blocked"`
}

type MessageDatas struct {
	db.GetFullMessageDatasRow
	ClientDatas
}