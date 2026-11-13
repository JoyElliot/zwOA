CREATE TABLE chat_conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    direct_key TEXT UNIQUE,
    creator_id INTEGER NOT NULL REFERENCES accounts(id),
    request_key TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_message_id INTEGER,
    UNIQUE (creator_id, request_key)
);

CREATE TABLE chat_members (
    conversation_id INTEGER NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    last_read_id INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (conversation_id, account_id)
);
CREATE INDEX chat_members_account ON chat_members(account_id, conversation_id);

CREATE TABLE chat_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL REFERENCES accounts(id),
    request_key TEXT NOT NULL,
    text TEXT NOT NULL,
    sent_at INTEGER NOT NULL,
    UNIQUE (conversation_id, sender_id, request_key)
);
CREATE INDEX chat_messages_history ON chat_messages(conversation_id, id);
