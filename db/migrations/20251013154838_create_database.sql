-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    provider_id TEXT,
    username TEXT NOT NULL UNIQUE CHECK (username <> ''),
    email TEXT NOT NULL UNIQUE CHECK (email <> ''),
    password TEXT,
    join_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT
);

CREATE TABLE user_relations (
    user_id UUID,
    relation_user_id UUID,
    type TEXT NOT NULL CHECK(type <> ''),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(user_id, relation_user_id, type),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (relation_user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE settings (
    user_id UUID,
    accepts_friend_reqs BOOLEAN NOT NULL DEFAULT TRUE,
    accepts_dm_reqs BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY(user_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE chatrooms (
    id INT8 GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title TEXT NOT NULL,
    creator_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (creator_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE user_joined_chatrooms (
    user_id UUID,
    chatroom_id INT8,
    join_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chatroom_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (chatroom_id) REFERENCES chatrooms (id) ON DELETE CASCADE
);

CREATE TABLE dms (
    id INT8 GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user1_id UUID NOT NULL,
    user2_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'not_accepted',
    creation_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user1_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user2_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE messages (
    id INT8 GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    text TEXT NOT NULL CHECK(text <> ''),
    user_id UUID NOT NULL,
    chatroom_id INT8,
    dm_id INT8,
    reply_id INT8,
    reply_status TEXT NOT NULL DEFAULT 'no_reply',
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (chatroom_id) REFERENCES chatrooms (id) ON DELETE CASCADE,
    FOREIGN KEY (dm_id) REFERENCES dms (id) ON DELETE CASCADE,
    FOREIGN KEY (reply_id) REFERENCES messages (id)
);

CREATE TABLE notifications (
    id INT8 GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    message TEXT NOT NULL CHECK(message <> ''),
    date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    link TEXT NOT NULL CHECK(link <> ''),
    type TEXT NOT NULL CHECK(type <> ''),
    read BOOLEAN NOT NULL DEFAULT FALSE,
    user_id UUID NOT NULL,
    notif_from TEXT NOT NULL CHECK(notif_from <> ''),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE auth_verification (
    id INT8 GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL,
    type TEXT NOT NULL CHECK(type <> ''),
    token TEXT NOT NULL CHECK(token <> ''),
    expiry TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
