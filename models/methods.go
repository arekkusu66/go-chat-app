package models


func (User) CheckUserRelations(targetUser User, relationType []User) bool {
	var m = make(map[string]bool)

	for _, relationUser := range relationType {
		m[relationUser.ID] = true
	}

	return m[targetUser.ID]
}


func (u User) AlreadyJoined(chatroom ChatRoom) bool {
	c, j  := make(map[uint]bool), make(map[uint]bool)

	for _, createdChat := range u.CreatedChats {
		c[createdChat.ID] = true
	}

	for _, joinedChat := range u.JoinedChats {
		j[joinedChat.ID] = true
	}

	return c[chatroom.ID] || j[chatroom.ID]
}


func (u User) AvailableChats() []ChatRoom {
	var notav = []uint{}

	for _, chatroom := range u.CreatedChats {
		notav = append(notav, chatroom.ID)
	}

	for _, chatroom := range u.JoinedChats {
		notav = append(notav, chatroom.ID)
	}

	var availableChats = []ChatRoom{}
	DB.Not(notav).Find(&availableChats)

	return availableChats
}


func (u User) GetTheOtherUser(dm DM) User {
	if len(dm.Users) == 1 {

		if dm.Users[0].ID != u.ID {
			return dm.Users[0]
		} else {
			return dm.RequestToUser[0]
		}

	} else if len(dm.Users) == 2 {
		
		if dm.Users[0].ID != u.ID {
			return dm.Users[0]
		} else {
			return dm.Users[1]
		}

	} else {
		return User{}
	}
}


func (u User) GetDMid(targetUser User) uint {
	var m = make(map[bool]uint)

	for _, dm := range u.DMS {
		m[dm.HasUser(targetUser)] = dm.ID
	}

	return m[true]
}


func (u User) UnreadNotifsCount() int64 {
	return DB.Model(&u).Where("read = ?", false).Association("Notifications").Count()
}


func (u User) AreThereUnreadNotifs() bool {
	if DB.Model(&u).Where("read = ?", false).Association("Notifications").Count() == 0 {
		return false
	} else {
		return true
	}
}


func (dm DM) HasUser(user User) bool {
	var (
		u = make(map[string]bool)
		d = make(map[uint]bool)
		i = make(map[uint]bool)
	)

	
	for _, usr := range dm.Users {
		u[usr.ID] = true
	}

	for _, dm := range user.DMRequests {
		d[dm.ID] = true
	}

	for _, dm := range user.IgnoredDMS {
		i[dm.ID] = true
	}


	return u[user.ID] || d[dm.ID] || i[dm.ID]
}


func (dm DM) LastMessage() Message {
	var message Message
	DB.Preload("User").Last(&message, "dm_id = ?", dm.ID)

	return message
}