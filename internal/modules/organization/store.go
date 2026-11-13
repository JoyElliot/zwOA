package organization

import (
	"context"
	"database/sql"
	"errors"

	"modernc.org/sqlite"
)

var (
	ErrInvalid      = errors.New("输入无效")
	ErrConflict     = errors.New("名称已存在，或引用的记录已变化")
	ErrStale        = errors.New("管理设置已更新，请刷新后重新编辑")
	ErrNotFound     = errors.New("记录不存在")
	ErrUnauthorized = errors.New("请先登录")
	ErrForbidden    = errors.New("没有此操作的权限")
	ErrPassword     = errors.New("当前密码不正确")
	ErrCredentials  = errors.New("账户或密码不正确")
	ErrTooMany      = errors.New("操作过于频繁，请稍后再试")
)

type Store struct {
	db        *sql.DB
	hashSlots chan struct{}
	attempts  loginAttempts
}

func New(db *sql.DB) *Store {
	return &Store{db: db, hashSlots: make(chan struct{}, 4)}
}

func (s *Store) transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		var constraint *sqlite.Error
		if errors.As(err, &constraint) && constraint.Code()&0xff == 19 {
			return ErrConflict
		}
		return err
	}
	return tx.Commit()
}

func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrNotFound
	}
	return err
}

func collect[T any](rows *sql.Rows, err error, scan func(*sql.Rows) (T, error)) ([]T, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []T{}
	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
