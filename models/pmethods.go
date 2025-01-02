package models

import (
	"fmt"
	"gochat/types"
	"net/http"
)


func (u *User) Add(targetUser *User, w http.ResponseWriter) {
	switch true {
		case u.CheckUserRelations(*u, targetUser.BlockedUsers):
			w.WriteHeader(http.StatusBadRequest)
			return

		case u.CheckUserRelations(*targetUser, u.BlockedUsers):
			http.Error(w, "you blocked this user", http.StatusBadRequest)
			return

		case u.CheckUserRelations(*targetUser, u.SentFriendReqs):
			http.Error(w, "you already sent a friend request to this user", http.StatusBadRequest)
			return
			
		case u.CheckUserRelations(*targetUser, u.Friends):
			http.Error(w, "this user is already in your friends list", http.StatusBadRequest)
			return
		
		default:
			u.SentFriendReqs = append(u.SentFriendReqs, *targetUser)
			targetUser.ReceivedFriendReqs = append(targetUser.ReceivedFriendReqs, *u)
		
			DB.Save(u) 
			DB.Save(targetUser)
		
			w.Write([]byte(fmt.Sprintf("<div id=\"friend-req-sent\"><h3>Friend request sent to %s</h3><button hx-post=\"/user/actionss?type=cancel&username=%s\" hx-trigger=\"click\" hx-target=\"#friend-req-sent\" hx-swap=\"outerHTML\">cancel</button></div>", targetUser.Username, targetUser.Username)))
	}
}


func (u *User) Cancel(targetUser *User, w http.ResponseWriter) {
	if u.CheckUserRelations(*targetUser, u.Friends) {
		w.Write([]byte(fmt.Sprintf("<div id=\"friends\"><h3>You are friend with %s</h3></div>", targetUser.Username)))
		return
	}

	if u.CheckUserRelations(*targetUser, u.SentFriendReqs) {
		DB.Model(u).Association("SentFriendReqs").Delete(targetUser)
		DB.Model(targetUser).Association("ReceivedFriendReqs").Delete(u)
		DB.Where("user_id = ? AND notif_from = ? AND type = ?", targetUser.ID, u.ID, types.FRIEND_REQ).Delete(&Notification{})
		
		var addAction = fmt.Sprintf("<button id=\"send-friend-req\" hx-post=\"/user/actionss?type=add&username=%s\" hx-trigger=\"click\" hx-target=\"#user-actions\" hx-swap=\"outerHTML\" onclick=\"sendNotif(this)\" data-type=\"friend-req\" data-target=\"%s\">send a friend request to %s</button>", targetUser.Username, targetUser.Username, targetUser.Username)
	
		var blockAction = fmt.Sprintf("<button hx-post=\"/user/actions?type=block&username=%s\" hx-trigger=\"click\" hx-target=\"#user-actions\" hx-swap=\"outerHTML\">block user</button>", targetUser.Username)
	
		w.Write([]byte(fmt.Sprintf("<div id=\"user-actions\">%s%s</div", addAction, blockAction)))
	}
}


func (u *User) Accept(targetUser *User, w http.ResponseWriter) {
	if u.CheckUserRelations(*targetUser, u.ReceivedFriendReqs) {
		DB.Model(u).Association("ReceivedFriendReqs").Delete(targetUser)
		DB.Model(targetUser).Association("SentFriendReqs").Delete(u)

		u.Friends = append(u.Friends, *targetUser)
		targetUser.Friends = append(targetUser.Friends, *u)

		DB.Save(u)
		DB.Save(targetUser)

		DB.Where("user_id = ? AND notif_from = ? AND type = ?", u.ID, targetUser.ID, types.FRIEND_REQ).Delete(&Notification{})

		w.Write([]byte(fmt.Sprintf("<div id=\"friends\"><h3>You are friend with %s</h3></div>", targetUser.Username)))
	}
}


func (u *User) Ignore(targetUser *User, w http.ResponseWriter) {
	if u.CheckUserRelations(*u, targetUser.SentFriendReqs) {
		DB.Model(u).Association("ReceivedFriendReqs").Delete(targetUser)
		DB.Where("user_id = ? AND notif_from = ? AND type = ?", u.ID, targetUser.ID, types.FRIEND_REQ).Delete(&Notification{})
		w.Write([]byte(fmt.Sprintf("You ignored %s's friend request", targetUser.Username)))
	}
}


func (u *User) Block(targetUser *User, w http.ResponseWriter) {
	if u.CheckUserRelations(*targetUser, u.Friends) {
		DB.Model(u).Association("Friends").Delete(targetUser)
		DB.Model(targetUser).Association("Friends").Delete(u)
	}

	if u.CheckUserRelations(*targetUser, u.DMedUsers) {
		if u.GetDMid(*targetUser) != 0 {
			var dm DM
			DB.First(&dm, u.GetDMid(*targetUser))
			DB.Model(u).Association("IgnoredDMS").Append(&dm)
		}
	}

	u.BlockedUsers = append(u.BlockedUsers, *targetUser)

	DB.Save(u)
	DB.Save(targetUser)

	w.Write([]byte(fmt.Sprintf("<div id=\"blocked-user\"><h3>You blocked %s</h3><button hx-post=\"/user/actions?type=unblock&username=%s\" hx-trigger=\"click\" hx-target=\"#blocked-user\" hx-swap=\"outerHTML\">unblock</button></div>", targetUser.Username, targetUser.Username)))
}


func (u *User) Unblock(targetUser *User, w http.ResponseWriter) {
	if u.CheckUserRelations(*targetUser, u.BlockedUsers) {

		if u.CheckUserRelations(*targetUser, u.DMedUsers) {
			if u.GetDMid(*targetUser) != 0 {
				var dm DM
				DB.First(&dm, u.GetDMid(*targetUser))
				DB.Model(u).Association("IgnoredDMS").Delete(&dm)
			}
		}

		DB.Model(u).Association("BlockedUsers").Delete(targetUser)

		var addAction = fmt.Sprintf("<button id=\"send-friend-req\" hx-post=\"/user/actions?action=add&username=%s\" hx-trigger=\"click\" hx-target=\"#user-actions\" hx-swap=\"outerHTML\">send friend requests to %s</button>", targetUser.Username, targetUser.Username)
	
		var blockAction = fmt.Sprintf("<button hx-post=\"/user/actions?action=block&username=%s\" hx-trigger=\"click\" hx-target=\"#user-actions\" hx-swap=\"outerHTML\">block user</button>", targetUser.Username)
	
		w.Write([]byte(fmt.Sprintf("<div id=\"user-actions\">%s%s</div", addAction, blockAction)))
	}
}


func (u *User) SendDM(targetUser *User, w http.ResponseWriter) {
	switch true {
		case u.CheckUserRelations(*u, targetUser.BlockedUsers):
			w.WriteHeader(http.StatusBadRequest)
			return

		case u.CheckUserRelations(*targetUser, u.BlockedUsers):
			http.Error(w, "you blocked this user", http.StatusBadRequest)
			return

		case u.CheckUserRelations(*u, targetUser.DMedUsers):
			http.Error(w, "you already messaged this user", http.StatusBadRequest)
			return

		default:
			var dm DM

			dm.Users = append(dm.Users, *u)
			dm.RequestToUser = append(dm.RequestToUser, *targetUser)
		
			DB.Create(&dm)
		
			u.DMedUsers = append(u.DMedUsers, *targetUser)
			targetUser.DMedUsers = append(targetUser.DMedUsers, *u)
		
			DB.Save(u)
			DB.Save(targetUser)
	}
}


func (u *User) AcceptDM(dm *DM, w http.ResponseWriter) {
	if u.CheckUserRelations(*u, dm.Users) {
		http.Error(w, "You already accepted this request", http.StatusBadRequest)
		return
	}

	if u.CheckUserRelations(*u, dm.RequestToUser) {
		DB.Model(u).Association("DMRequests").Delete(dm)
		DB.Model(dm).Association("RequestToUser").Delete(u)
		dm.Users = append(dm.Users, *u)
		DB.Save(dm)
	}
}


func (u *User) IgnoreDM(dm *DM) {
	if u.CheckUserRelations(*u, dm.RequestToUser) {	
		DB.Model(u).Association("DMRequests").Delete(dm)
		u.IgnoredDMS = append(u.IgnoredDMS, *dm)
		DB.Save(u)
	}
}