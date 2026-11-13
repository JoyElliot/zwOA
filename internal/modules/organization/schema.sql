CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    super_admin INTEGER NOT NULL DEFAULT 0 CHECK (super_admin IN (0, 1)),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    must_change_password INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1))
);
CREATE UNIQUE INDEX IF NOT EXISTS one_super_admin ON accounts(super_admin) WHERE super_admin = 1;

CREATE TABLE IF NOT EXISTS sessions (
    token_hash BLOB PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_account ON sessions(account_id);
CREATE INDEX IF NOT EXISTS sessions_expiry ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS permission_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS group_permissions (
    group_id INTEGER NOT NULL REFERENCES permission_groups(id) ON DELETE CASCADE,
    permission TEXT NOT NULL,
    PRIMARY KEY (group_id, permission)
);
CREATE TABLE IF NOT EXISTS account_groups (
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES permission_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, group_id)
);

CREATE TABLE IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL CHECK (kind IN ('team', 'project')),
    description TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS team_members (
    team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    is_admin INTEGER NOT NULL CHECK (is_admin IN (0, 1)),
    PRIMARY KEY (team_id, account_id)
);
CREATE INDEX IF NOT EXISTS team_members_account ON team_members(account_id);
CREATE TABLE IF NOT EXISTS module_admins (
    module TEXT NOT NULL,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    PRIMARY KEY (module, account_id)
);
CREATE INDEX IF NOT EXISTS module_admins_account ON module_admins(account_id);

CREATE TABLE IF NOT EXISTS administration_revision (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    revision INTEGER NOT NULL
);
INSERT INTO administration_revision(id, revision) VALUES (1, 0) ON CONFLICT(id) DO NOTHING;
