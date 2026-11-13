package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/JoyElliot/zwOA/internal/platform/sqlite"
)

func TestOpenClearsLegacyPasswordChangeRequirement(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "zwoa.db")
	db, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := organization.New(db)
	if err := store.Bootstrap(ctx, "test-admin", "管理员", "bootstrap-password"); err != nil {
		t.Fatal(err)
	}
	token, err := store.Login(ctx, "test-admin", "bootstrap-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE accounts SET must_change_password = 1; PRAGMA user_version = 2`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	account, err := organization.New(upgraded).Session(ctx, token)
	if err != nil || !account.SuperAdmin || account.MustChangePassword {
		t.Fatalf("旧账户或会话未恢复正常访问：%+v, %v", account, err)
	}
}
