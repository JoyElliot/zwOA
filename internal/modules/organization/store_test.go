package organization

import (
	"path/filepath"
	"testing"
)

func TestOpenClearsLegacyPasswordChangeRequirement(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "zwoa.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Bootstrap(ctx, "test-admin", "管理员", "bootstrap-password"); err != nil {
		t.Fatal(err)
	}
	token, err := store.Login(ctx, "test-admin", "bootstrap-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE accounts SET must_change_password = 1; PRAGMA user_version = 2`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	account, err := upgraded.Session(ctx, token)
	if err != nil || !account.SuperAdmin || account.MustChangePassword {
		t.Fatalf("旧账户或会话未恢复正常访问：%+v, %v", account, err)
	}
}
