package db

import (
	"context"
	"gochat/app/types"

	"github.com/google/uuid"
)


func (u *User) UserRelations(ctx context.Context, Query *Queries, kind types.UserRelation) []User {
	switch kind {
	case types.FRIEND:
		friends, _ := Query.GetAllRelations(ctx, GetAllRelationsParams{
			UserID: u.ID,
			Kind: types.FRIEND,
		})

		return friends

	case types.RECEIVED_FRIEND_REQ:
		receivedReqs, _ := Query.GetAllRelations(ctx, GetAllRelationsParams{
			UserID: u.ID,
			Kind: types.RECEIVED_FRIEND_REQ,
		})

		return receivedReqs

	case types.SENT_FRIEND_REQ:
		sentRequests, _ := Query.GetAllRelations(ctx, GetAllRelationsParams{
			UserID: u.ID,
			Kind: types.SENT_FRIEND_REQ,
		})

		return sentRequests

	case types.BLOCKED_USER:
		blockedUsers, _ := Query.GetAllRelations(ctx, GetAllRelationsParams{
			UserID: u.ID,
			Kind: types.BLOCKED_USER,
		})

		return blockedUsers

	default:
		return []User{}
	}
}


func (u *User) CheckUserRelation(ctx context.Context, Query *Queries, Kind types.UserRelation, targetId uuid.UUID) bool {
	switch Kind {
	case types.FRIEND:
		isFriend, _ := Query.CheckRelation(ctx, CheckRelationParams{
			UserID: u.ID,
			RelationUserID: targetId,
			Kind: types.FRIEND,
		})

		return isFriend

	case types.RECEIVED_FRIEND_REQ:

		isReceivedRequest, _ := Query.CheckRelation(ctx, CheckRelationParams{
			UserID: u.ID,
			RelationUserID: targetId,
			Kind: types.RECEIVED_FRIEND_REQ,
		})

		return isReceivedRequest

	case types.SENT_FRIEND_REQ:
		isSentRequest, _ := Query.CheckRelation(ctx, CheckRelationParams{
			UserID: u.ID,
			RelationUserID: targetId,
			Kind: types.SENT_FRIEND_REQ,
		})

		return isSentRequest

	case types.BLOCKED_USER:
		isBlocked, _ := Query.CheckRelation(ctx, CheckRelationParams{
			UserID: u.ID,
			RelationUserID: targetId,
			Kind: types.BLOCKED_USER,
		})

		return isBlocked

	default:
		return false
	}
}