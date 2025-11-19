-- name: CreateUser :exec
INSERT INTO users (
    id,
    provider_id,
    username, 
    email, 
    verified,
    password
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetUserById :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByProviderId :one
SELECT * FROM users WHERE provider_id = $1;

-- name: GetUserByName :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByMessage :one
SELECT u.* FROM users u INNER JOIN messages m ON m.user_id = u.id WHERE m.id = $1;

-- name: GetUserChatroomDatas :one
SELECT u.description, ujc.join_date
FROM users u
INNER JOIN user_joined_chatrooms ujc ON ujc.user_id = u.id
WHERE u.id = $1 AND ujc.chatroom_id = $2;

-- name: UpdateUser :exec
UPDATE users 
SET username = COALESCE(NULLIF(sqlc.arg(username)::TEXT, ''), username),
    password = COALESCE(NULLIF(sqlc.arg(password)::TEXT, ''), password),
    verified = CASE 
                WHEN sqlc.arg(verified)::BOOL = TRUE AND verified = FALSE THEN TRUE 
                WHEN sqlc.arg(verified)::BOOL = FALSE AND verified = TRUE THEN TRUE
                WHEN sqlc.arg(verified)::BOOL = TRUE AND verified = TRUE THEN TRUE 
                ELSE FALSE END,
    description = COALESCE(NULLIF(sqlc.arg(description)::TEXT, ''), description)
WHERE id = $1;

-- name: CheckUsernameAvailability :one
SELECT EXISTS ( SELECT 1 FROM users WHERE username = $1 );

-- name: CheckUserByProviderId :one
SELECT EXISTS ( SELECT 1 FROM users WHERE provider_id = $1 );

-- name: CreateRelation :exec
INSERT INTO user_relations (user_id, relation_user_id, type) VALUES ($1, $2, $3);

-- name: GetAllRelations :many
SELECT u.*
FROM users u
INNER JOIN user_relations ur ON ur.relation_user_id = u.id
WHERE ur.user_id = $1 AND type = $2;

-- name: DeleteRelation :exec
DELETE FROM user_relations WHERE user_id = $1 AND relation_user_id = $2 AND type = $3;

-- name: DeleteMutualRelation :exec
DELETE FROM user_relations
WHERE type = $1 AND (
    (user_id = $2 AND relation_user_id = $3) OR (user_id = $3 AND relation_user_id = $2)
);

-- name: CheckRelation :one
SELECT EXISTS ( SELECT 1 FROM user_relations WHERE user_id = $1 AND relation_user_id = $2 AND type = $3 );

-- name: CheckMutualRelation :one
SELECT EXISTS (
    SELECT 1
    FROM user_relations
    WHERE type = $1 AND ( 
        (user_id = $2 AND relation_user_id = $3) OR (user_id = $3 AND relation_user_id = $2)
    )
);

-- name: CreateAuthVerification :exec
INSERT INTO auth_verification (user_id, type, token, expiry) VALUES ($1, $2, $3, $4);

-- name: GetAuthVerification :one
SELECT * FROM auth_verification WHERE user_id = $1 AND type = $2;

-- name: DeleteAuthVerification :exec
DELETE FROM auth_verification WHERE user_id = $1 AND type = $2;

-- name: CheckAuthVerification :one
SELECT EXISTS ( SELECT 1 FROM auth_verification WHERE user_id = $1 );

-- name: InsertDefaultSettings :exec
INSERT INTO settings (user_id) VALUES ($1);

-- name: GetSettings :one
SELECT * FROM settings WHERE user_id = $1;

-- name: UpdateSettings :exec
UPDATE settings
SET
    accepts_friend_reqs = CASE
        WHEN sqlc.arg(accepts_friend_reqs)::TEXT = 'yes' THEN true
        WHEN sqlc.arg(accepts_friend_reqs)::TEXT = 'no' THEN false END,
    accepts_dm_reqs = CASE
        WHEN sqlc.arg(accepts_dm_reqs)::TEXT = 'yes' THEN true
        WHEN sqlc.arg(accepts_dm_reqs)::TEXT = 'no' THEN false END
WHERE user_id = @user_id;



-- name: CreateChatroom :one
INSERT INTO chatrooms (title, creator_id) VALUES ($1, $2) RETURNING id;

-- name: GetChatroom :one
SELECT * FROM chatrooms WHERE id = $1;

-- name: CheckChatroomExistence :one
SELECT EXISTS ( SELECT 1 FROM chatrooms WHERE id = $1 );

-- name: JoinChatroom :exec
INSERT INTO user_joined_chatrooms (user_id, chatroom_id) VALUES ($1, $2);

-- name: LeaveChatroom :exec
DELETE FROM user_joined_chatrooms WHERE user_id = $1 AND chatroom_id = $2;

-- name: CheckIfAlreadyJoined :one
SELECT EXISTS (
    SELECT 1
    FROM chatrooms c
    INNER JOIN user_joined_chatrooms ujc ON ujc.chatroom_id = $1 AND ujc.user_id = $2
);

-- name: GetCreator :one
SELECT u.*
FROM users u
INNER JOIN chatrooms c ON c.creator_id = u.id
WHERE c.id = $1;

-- name: GetAllJoinedUsers :many
SELECT u.*
FROM chatrooms c
INNER JOIN user_joined_chatrooms ujc ON ujc.chat_id = c.id
INNER JOIN users u ON u.id = ujc.user_id
WHERE c.id = $1;

-- name: GetUsersCount :one
SELECT COUNT(*)
FROM chatrooms c
INNER JOIN user_joined_chatrooms ujc ON ujc.chatroom_id = c.id
WHERE c.id = $1;

-- name: GetJoinDate :one
SELECT ujc.join_date
FROM user_joined_chatrooms ujc
WHERE ujc.chatroom_id = $1 AND ujc.user_id = $2;

-- name: GetChatroomMessages :many
SELECT m.*
FROM messages m
INNER JOIN chatrooms c ON c.id = m.chatroom_id
WHERE c.id = $1
LIMIT $2 OFFSET $3; 

-- name: GetCreatedChatrooms :many
SELECT c.*
FROM chatrooms c
INNER JOIN users u ON u.id = c.creator_id
WHERE u.id = $1;

-- name: GetJoinedChatrooms :many
SELECT c.*
FROM chatrooms c
INNER JOIN user_joined_chatrooms ujc ON ujc.chatroom_id = c.id
WHERE ujc.user_id = $1 AND ujc.user_id <> c.creator_id;

-- name: GetCreatedChatroomsCount :one
SELECT COUNT(*)
FROM chatrooms c
INNER JOIN users u ON u.id = c.creator_id
WHERE u.id = $1;

-- name: GetJoinedChatroomsCount :one
SELECT COUNT(*)
FROM chatrooms c
INNER JOIN user_joined_chatrooms ujc ON ujc.chatroom_id = c.id
WHERE ujc.user_id = $1;

-- name: GetAvailableChatrooms :many
SELECT c.*
FROM chatrooms c
WHERE NOT EXISTS (
	SELECT 1
	FROM user_joined_chatrooms ujc
	WHERE ujc.chatroom_id = c.id AND ujc.user_id = $1
) LIMIT $2 OFFSET $3;

-- name: GetCommonChatrooms :many
SELECT c.*
FROM chatrooms c
INNER JOIN user_joined_chatrooms ujc1 ON ujc1.chatroom_id = c.id
INNER JOIN user_joined_chatrooms ujc2 ON ujc2.chatroom_id = c.id
WHERE ujc1.user_id = $1 AND ujc2.user_id = $2;

-- name: GetAllMessages :many
SELECT m.*
FROM messages m
INNER JOIN chatrooms c ON c.id = m.chatroom_id
WHERE c.id = $1;



-- name: CreateDM :exec
INSERT INTO dms (user1_id, user2_id) VALUES ($1, $2);

-- name: GetDMById :one
SELECT * FROM dms WHERE id = $1;

-- name: GetDMWithUser :one
SELECT * FROM dms WHERE id = $1 AND (user1_id = $2 OR user2_id = $2);

-- name: GetDMWithBothUsers :one
SELECT * FROM dms WHERE (user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1);

-- name: GetAllDMsWithStatus :many
SELECT * FROM dms WHERE status = $1 AND (user1_id = $2 OR user2_id = $2);

-- name: CheckIfUserIsInDM :one
SELECT EXISTS (
	SELECT 1
	FROM dms
	WHERE id = $1 AND (user1_id = $2 OR user2_id = $2)
);

-- name: CheckIfUserDMedUser :one
SELECT EXISTS (
    SELECT 1
    FROM dms
    WHERE (user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1)
);

-- name: UpdateDM :exec
UPDATE dms SET status = $1 WHERE id = $2;

-- name: GetDMmessages :many
SELECT m.*
FROM messages m
WHERE m.dm_id = $1;

-- name: GetTheOtherDMuser :one
SELECT u.*
FROM dms
JOIN users u ON u.id = CASE
    WHEN dms.user1_id = $1 THEN dms.user2_id
    ELSE dms.user1_id
END
WHERE dms.id = $2 AND (dms.user1_id = $1 OR dms.user2_id = $1);

-- name: GetDMLastMessage :one
SELECT m.*
FROM messages m 
INNER JOIN dms ON dms.id = m.dm_id
WHERE dms.id = $1 
ORDER BY m.id DESC
LIMIT 1;



-- name: CreateMessage :one
INSERT INTO messages (
    text,
    user_id,
    chatroom_id,
    dm_id,
    reply_id,
    reply_status
) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: GetMessageById :one
SELECT * FROM messages WHERE id = $1;

-- name: GetFullMessageDatas :one
SELECT
    c.id AS chatroom_id, c.creator_id AS creator_id, 
    d.id AS dm_id,
    u.id AS user_id, u.username, 
    m.id, m.text, m.date, m.reply_id, m.reply_status,
    r.user_id AS reply_user_id, r.text AS reply_text
FROM messages m
INNER JOIN users u ON u.id = m.user_id
LEFT JOIN chatrooms c ON c.id = m.chatroom_id
LEFT JOIN dms d ON d.id = m.dm_id
LEFT JOIN messages r ON r.id = m.reply_id
WHERE m.id = $1;

-- name: UpdateReply :exec
UPDATE messages SET reply_id = NULL, reply_status = $1 WHERE reply_id = $2;

-- name: DeleteMessage :exec
DELETE FROM messages WHERE id = $1;

-- name: CheckIfMessageIsReply :one
SELECT EXISTS (
	SELECT 1
	FROM messages m
	LEFT JOIN messages r ON r.id = m.reply_id
	WHERE m.id = $1 AND r.user_id = $2
);

-- name: CheckMessageUserRelation :one
SELECT EXISTS (
    SELECT 1
    FROM messages m
    INNER JOIN user_relations ur ON ur.relation_user_id = m.user_id
    WHERE ur.user_id = $1 AND m.id = $2 AND type = $3
);



-- name: CreateNotification :exec
INSERT INTO notifications (
    message,
    link,
    type,
    user_id,
    notif_from
) VALUES ($1, $2, $3, $4, $5);

-- name: GetNoficationById :one
SELECT * FROM notifications WHERE id = $1;

-- name: GetAllNotifications :many
SELECT * FROM notifications WHERE user_id = $1;

-- name: DeleteAllNotifications :exec
DELETE FROM notifications WHERE id = $1;

-- name: DeleteAllNotificationsByType :exec
DELETE FROM notifications WHERE user_id = $1 AND type = $2;

-- name: ReadNotification :exec
UPDATE notifications SET read = true WHERE id = $1;

-- name: AreThereUnreadNotifications :one
SELECT EXISTS ( SELECT 1 FROM notifications WHERE user_id = $1 AND read = false );

-- name: UnreadNotificationsCount :one
SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false;

-- name: DeallocateAll :exec
DEALLOCATE ALL;