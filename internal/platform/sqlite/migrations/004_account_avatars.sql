CREATE TABLE IF NOT EXISTS account_avatars (
    account_id INTEGER PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    version TEXT NOT NULL UNIQUE,
    jpeg BLOB NOT NULL
);

PRAGMA user_version = 4;
