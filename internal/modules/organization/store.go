package organization

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

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

func Open(ctx context.Context, path string) (*Store, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(path)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	uri := url.URL{Scheme: "file", Path: uriPath, RawQuery: "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db, hashSlots: make(chan struct{}, 4)}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var version int
		if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
			return err
		}
		switch version {
		case 0:
			if _, err := tx.ExecContext(ctx, schema); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `PRAGMA user_version = 1`)
			return err
		case 1:
			return nil
		default:
			return fmt.Errorf("无法读取数据库版本 %d", version)
		}
	})
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
