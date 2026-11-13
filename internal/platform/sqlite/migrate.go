package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed migrations/001_organization.sql
var organizationSchema string

//go:embed migrations/002_chat.sql
var chatSchema string

//go:embed migrations/003_clear_password_change.sql
var clearPasswordChange string

//go:embed migrations/004_account_avatars.sql
var accountAvatars string

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var version int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version == 0 {
		if _, err := tx.ExecContext(ctx, organizationSchema); err != nil {
			return err
		}
		version = 1
	}
	switch version {
	case 1:
		if _, err := tx.ExecContext(ctx, chatSchema); err != nil {
			return err
		}
		fallthrough
	case 2:
		if _, err := tx.ExecContext(ctx, clearPasswordChange); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 3`); err != nil {
			return err
		}
		fallthrough
	case 3:
		if _, err := tx.ExecContext(ctx, accountAvatars); err != nil {
			return err
		}
	case 4:
	default:
		return fmt.Errorf("无法读取数据库版本 %d", version)
	}
	return tx.Commit()
}
