package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)


func CreateUser(ctx context.Context, userParams *CreateUserParams) error {
	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	if err := queryTx.CreateUser(ctx, *userParams); err != nil {
		return err
	}
	
	if !userParams.Verified {
		if err := queryTx.CreateAuthVerification(ctx, CreateAuthVerificationParams{
			UserID: userParams.ID,
			Type: types.EMAIL_VERIFICATION,
			Token: utils.GenerateToken(),
			Expiry: time.Now().Add(time.Hour),
		}); err != nil {
			log.Println("error creating an auth verification", err)
			return err
		}

		if err := queryTx.CreateNotification(ctx, CreateNotificationParams{
			UserID: userParams.ID,
			NotifFrom: "app",
			Message: "You have succesfully created your account, but you need to verify your email",
			Link: "/email/verification/send",
			Type: string(types.ACC_VERIFY_NEED),
		}); err != nil {
			log.Println("error creating a notification", err)
			return err
		}
	}

	if err := queryTx.InsertDefaultSettings(ctx, userParams.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}


func AddFriend(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	isBlocked, err := Query.CheckMutualRelation(ctx, CheckMutualRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	})

	if err != nil {
		return http.StatusBadRequest, errors.New("couldnt check the relation type with this user")
	}

	if isBlocked {
		return http.StatusBadRequest, errors.New("you blocked this user or the user blocked you")
	}

	settings, err := Query.GetSettings(ctx, targetId)

	if err != nil {
		return http.StatusBadRequest, errors.New("couldnt check the relation type with this user")
	}

	if !settings.AcceptsFriendReqs {
		return http.StatusBadRequest, errors.New("this user doesnt accept friend requests")
	}

	isFriend, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.FRIEND,
	})

	if err != nil {
		return http.StatusBadRequest, errors.New("couldnt check the relation type with this user")
	}

	if isFriend {
		return http.StatusBadRequest, errors.New("you are already friend with this user!")
	}

	sentFriendReq, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.SENT_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusBadRequest, errors.New("couldnt check the relation type with this user")
	}

	if sentFriendReq {
		return http.StatusBadRequest, errors.New("you already sent a friend request to this user")
	}

	receivedFriendReq, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.RECEIVED_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusBadRequest, errors.New("couldnt check the relation type with this user")
	}

	if receivedFriendReq {
		return http.StatusBadRequest, errors.New("you already received a friend request from this user")
	}

	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	if err := queryTx.CreateRelation(ctx, CreateRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.SENT_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt send a friend request to this user")
	}

	if err := queryTx.CreateRelation(ctx, CreateRelationParams{
		UserID: targetId,
		RelationUserID: userId,
		Type: types.RECEIVED_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt send a friend request to this user")
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	return 0, nil
}


func CancelFriendRequest(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	sentFriendReq, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.SENT_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if !sentFriendReq {
		return http.StatusBadRequest, errors.New("you havent send a friend request to this user")
	}

	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	if err := queryTx.DeleteRelation(ctx, DeleteRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.SENT_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt cancel the friend request")
	}

	if err := queryTx.DeleteRelation(ctx, DeleteRelationParams{
		UserID: targetId,
		RelationUserID: userId,
		Type: types.RECEIVED_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt cancel the friend request")
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	return 0, nil
}


func AcceptFriendRequest(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	isBlocked, err := Query.CheckMutualRelation(ctx, CheckMutualRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if isBlocked {
		return http.StatusBadRequest, errors.New("you blocked this user or the user blocked you")
	}

	receivedFriendReq, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.RECEIVED_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if !receivedFriendReq {
		return http.StatusBadRequest, errors.New("you havent received a friend request from this user")
	}

	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	if err := queryTx.CreateRelation(ctx, CreateRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.FRIEND,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt accept the friend request")
	}

	if err := queryTx.CreateRelation(ctx, CreateRelationParams{
		UserID: targetId,
		RelationUserID: userId,
		Type: types.FRIEND,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt accept the friend request")
	}

	if err := queryTx.DeleteRelation(ctx, DeleteRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.RECEIVED_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt accept the friend request")
	}

	if err := queryTx.DeleteRelation(ctx, DeleteRelationParams{
		UserID: targetId,
		RelationUserID: userId,
		Type: types.SENT_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt accept the friend request")
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	return 0, nil
}


func IgnoreFriendRequest(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	receivedFriendReq, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.RECEIVED_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if !receivedFriendReq {
		return http.StatusBadRequest, errors.New("you havent received a friend request from this user")
	}

	if err := Query.DeleteRelation(ctx, DeleteRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.RECEIVED_FRIEND_REQ,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt ignore the friend request")
	}

	return 0, nil
}


func BlockUser(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	isAlreadyBlocked, err := queryTx.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if isAlreadyBlocked {
		return http.StatusBadRequest, errors.New("you already blocked this user")
	}

	isFriend, err := queryTx.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.FRIEND,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if isFriend {
		if err := queryTx.DeleteMutualRelation(ctx, DeleteMutualRelationParams{
			UserID: userId,
			RelationUserID: targetId,
			Type: types.FRIEND,
		}); err != nil {
			return http.StatusInternalServerError, errors.New("coulnt block the user")
		}
	}

	sentFriendReq, err := queryTx.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.SENT_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if sentFriendReq {
		if err := queryTx.DeleteRelation(ctx, DeleteRelationParams{
			UserID: userId,
			RelationUserID: targetId,
			Type: types.SENT_FRIEND_REQ,
		}); err != nil {
			return http.StatusInternalServerError, errors.New("couldnt block the user")
		}
	}

	receivedFriendReq, err := queryTx.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.RECEIVED_FRIEND_REQ,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if receivedFriendReq {
		if err := queryTx.DeleteRelation(ctx, DeleteRelationParams{
			UserID: userId,
			RelationUserID: targetId,
			Type: types.RECEIVED_FRIEND_REQ,
		}); err != nil {
			return http.StatusInternalServerError, errors.New("couldnt block the user")
		}
	}

	if err := queryTx.CreateRelation(ctx, CreateRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt block the user")
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	return 0, nil
}


func UnblockUser(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	isBlocked, err := Query.CheckRelation(ctx, CheckRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if !isBlocked {
		return http.StatusBadRequest, errors.New("you didnt block this user")
	}

	if err := Query.DeleteRelation(ctx, DeleteRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt unblock the user")
	}

	return 0, nil
}


func CreateChatroom(ctx context.Context, userId uuid.UUID, title string) error {
	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	id, err := queryTx.CreateChatroom(ctx, CreateChatroomParams{
		CreatorID: userId,
		Title: title,
	})

	if err != nil {
		return err
	}

	if err := queryTx.JoinChatroom(ctx, JoinChatroomParams{
		ChatroomID: id,
		UserID: userId,
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}


func SendDM(ctx context.Context, userId, targetId uuid.UUID) (int, error) {
	isBlocked, err := Query.CheckMutualRelation(ctx, CheckMutualRelationParams{
		UserID: userId,
		RelationUserID: targetId,
		Type: types.BLOCKED_USER,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if isBlocked {
		return http.StatusBadRequest, errors.New("you blocked this user or the user blocked you")
	}

	settings, err := Query.GetSettings(ctx, targetId)

	if err != nil {
		return http.StatusBadRequest, errors.New("couldnt check the relation type with this user")
	}

	if !settings.AcceptsDmReqs {
		return http.StatusBadRequest, errors.New("this user doesnt accept dm requests")
	}

	hasDmed, err := Query.CheckIfUserDMedUser(ctx, CheckIfUserDMedUserParams{
		User1ID: userId,
		User2ID: targetId,
	})

	if err != nil {
		return http.StatusInternalServerError, errors.New("couldnt check the relation type with this user")
	}

	if hasDmed {
		return http.StatusBadRequest, errors.New("you already started a conversation with this user")
	}

	fmt.Println(hasDmed)

	if err := Query.CreateDM(ctx, CreateDMParams{
		User1ID: userId,
		User2ID: targetId,
	}); err != nil {
		return http.StatusInternalServerError, errors.New("couldnt send a dm request to the user")
	}

	return 0, nil
}


func DeleteMessage(ctx context.Context, id int64) error {
	tx, err := DB.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queryTx = Query.WithTx(tx)

	if err := queryTx.UpdateReply(ctx, UpdateReplyParams{
		ReplyID: sql.NullInt64{Int64: id, Valid: true},
		ReplyStatus: "deleted",
	}); err != nil {
		return err
	}

	if err := queryTx.DeleteMessage(ctx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}


// func IsMessageAReply(ctx context.Context, replyId int64, userId uuid.UUID) bool {
// 	if replyId == 0 {
// 		return false
// 	}

// 	replyToMessage, err := Query.GetMessageById(ctx, replyId)

// 	if err != nil {
// 		return false
// 	}

// 	fmt.Println("reply to user id:", replyToMessage.UserID)

// 	if replyToMessage.UserID == userId {
// 		return true
// 	}

// 	return false
// }