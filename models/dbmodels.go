package models

import (
	"database/sql"
	"time"

	"golang.org/x/oauth2"
)


type User struct {
	ID							string				`gorm:"primaryKey" json:"id"`
	Username					string				`gorm:"unique" json:"username"`
	Email						string				`gorm:"unique" json:"email"`
	Password					sql.NullString		`json:"-"`

	EmailVerificationID			*uint				`json:"-"`
	EmailVerification			*AuthVerification	`gorm:"foreignKey:EmailVerificationID" json:"-"`

	PasswordVerificationID		*uint				`json:"-"`
	PasswordVerification		*AuthVerification	`gorm:"foreignKey:PasswordVerificationID" json:"-"`

	Verified					bool				`json:"-"`
	Description					sql.NullString		`json:"description"`
	Joined						time.Time

	Settings					Setting

	SentFriendReqs				[]User				`gorm:"many2many:sent_friend_requests"`
	ReceivedFriendReqs			[]User				`gorm:"many2many:received_friend_requests"`
	Friends						[]User				`gorm:"many2many:user_friends"`
	BlockedUsers				[]User				`gorm:"many2many:blocked_users"`
	DMedUsers					[]User				`gorm:"many2many:dmed_users"`

	DMS							[]DM				`gorm:"many2many:user_dms"`
	DMRequests					[]DM				`gorm:"many2many:dm_requests"`
	IgnoredDMS					[]DM				`gorm:"many2many:ignored_dms"`

	Messages					[]Message

	Notifications				[]Notification

	CreatedChats				[]ChatRoom			`gorm:"many2many:user_created_chats"`
	JoinedChats					[]ChatRoom			`gorm:"many2many:user_joined_chats"`
}


type SessionData struct {
	ID							string				`gorm:"primaryKey"`
	*oauth2.Token
}


type Message struct {
	ID							uint				`gorm:"primaryKey" json:"id"`
	Date						time.Time			`json:"date"`
	Text						string				`json:"text"`

	ReplyID						uint				`json:"replyId"`
	Reply						*Message			`json:"reply"`
	ReplyStatus					string				`json:"replyStatus"`
	
	UserID						string				`json:"userId"`
	User						User				`gorm:"foreignKey:UserID" json:"user"`
	ChatOp						sql.NullString		`json:"chatOp"`

	DMID						uint				`json:"dmId"`
	DM							DM					`gorm:"foreignKey:DMID"`

	ChatRoomID					uint				`json:"chatRoomId"`
	ChatRoom					ChatRoom			`gorm:"foreignKey:ChatRoomID"`
}


type ChatRoom struct {
	ID							uint				`gorm:"primaryKey"`
	Title						string				`json:"title"`

	CreatedBy					string
	CreatedByName				string

	Messages					[]Message
	JoinedUsers					[]User				`gorm:"many2many:user_joined_chats"`
}


type DM struct {
	ID							uint				`gorm:"primaryKey"`

	Users						[]User				`gorm:"many2many:user_dms"`
	IgnoredBy					[]User				`gorm:"many2many:ignored_dms"`
	RequestToUser				[]User				`gorm:"many2many:dm_requests"`
	
	Messages					[]Message
}


type Notification struct {
	ID							uint				`gorm:"primaryKey" json:"id"`

	Message						string				`json:"message"`
	Date						time.Time			`json:"date"`
	Link						string				`json:"link"`

	Type						string				`json:"type"`
	Read						bool				`json:"read"`

	UserID						string
	User						User				`gorm:"foreignKey:UserID" json:"user"`

	NotifFrom					string				`json:"from"`
}


type AuthVerification struct {
	ID							uint				`gorm:"primaryKey"`

	Type						string
	Token						string
	Expiry						time.Time

	UserID						string
	User						User				`gorm:"foreignKey:UserID"`
}


type Setting struct {
	ID							uint				`gorm:"primaryKey"`

	UserID						*string
	User						*User				`gorm:"foreignKey:UserID"`

	AcceptsFriendReqs			bool
	AcceptsDMReqs				bool
}