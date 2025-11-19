package types


type (
	UserRelation string
	Verification string
	Setting		 string
	ChatRoom 	 string
	Notif 		 string
	DM 			 string
	WSMessage 	 string
)

const (
	SENT_FRIEND_REQ 	UserRelation = "sent_friend_request"
	RECEIVED_FRIEND_REQ UserRelation = "received_friend_request"
	FRIEND 				UserRelation = "friend"
	BLOCKED_USER 		UserRelation = "blocked_user"
)

const (
	ACC_CREATED			Verification = "account_created"
	ACC_VERIFY_NEED		Verification = "need_to_verify_account"
	ACC_VERIFIED		Verification = "account_verified"
	EMAIL_VERIFICATION  Verification = "email_verification"
	PASSWORD_RESET 	    Verification = "password_reset"
	PASSWORD_NEW		Verification = "new_password"
)

const (
	ACCEPTS_FRIEND_REQS Setting = "accepts_friend_reqs"
	ACCEPTS_DM_REQS		Setting = "accepts_dm_reqs"
)

const (
	JOINED_CHAT  ChatRoom = "joined_chatroom"
	CREATED_CHAT ChatRoom = "created_chatroom"
)

const (
	ACCEPTED_DM 	DM = "accepted"
	NOT_ACCEPTED_DM DM = "not_accepted"
)

const (
	MSG	  	WSMessage = "MSG"
	DEL	  	WSMessage = "DEL"
	NOTIF 	WSMessage = "NOTIF"
)