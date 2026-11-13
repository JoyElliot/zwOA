package organization

import (
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
)

type Module struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

var modules = []Module{
	{Key: "chat", Name: "消息", Permission: chat.Permission},
	{Key: "crm", Name: "客户管理", Permission: "crm.access"},
	{Key: "approvals", Name: "办公审批", Permission: "approvals.access"},
	{Key: "organization", Name: "组织与权限", Permission: "organization.access"},
}

type ManagedAccount struct {
	Account
	GroupIDs []int64 `json:"groupIds"`
}

type AccountInput struct {
	Username string  `json:"username"`
	Name     string  `json:"name"`
	Password string  `json:"password"`
	Active   bool    `json:"active"`
	GroupIDs []int64 `json:"groupIds"`
}

type PermissionGroup struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type Team struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	MemberIDs   []int64 `json:"memberIds"`
	AdminIDs    []int64 `json:"adminIds"`
}

type ManagedModule struct {
	Module
	AdminIDs []int64 `json:"adminIds"`
}

type Administration struct {
	Revision int64             `json:"revision"`
	Accounts []ManagedAccount  `json:"accounts"`
	Groups   []PermissionGroup `json:"groups"`
	Teams    []Team            `json:"teams"`
	Modules  []ManagedModule   `json:"modules"`
}

func (s *Store) Administration(ctx context.Context) (Administration, error) {
	var data Administration
	for _, module := range modules {
		data.Modules = append(data.Modules, ManagedModule{Module: module, AdminIDs: []int64{}})
	}
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM administration_revision WHERE id = 1`).Scan(&data.Revision); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT `+accountColumns+` FROM accounts ORDER BY super_admin DESC, username`)
		data.Accounts, err = collect(rows, err, func(row *sql.Rows) (ManagedAccount, error) {
			account := ManagedAccount{GroupIDs: []int64{}}
			err := account.scan(row)
			return account, err
		})
		if err != nil {
			return err
		}
		accounts := make(map[int64]*ManagedAccount, len(data.Accounts))
		for i := range data.Accounts {
			accounts[data.Accounts[i].ID] = &data.Accounts[i]
		}
		if err := eachRow(ctx, tx, `SELECT account_id, group_id FROM account_groups ORDER BY group_id`, func(row *sql.Rows) error {
			var accountID, groupID int64
			if err := row.Scan(&accountID, &groupID); err != nil {
				return err
			}
			accounts[accountID].GroupIDs = append(accounts[accountID].GroupIDs, groupID)
			return nil
		}); err != nil {
			return err
		}

		rows, err = tx.QueryContext(ctx, `SELECT id, name, description FROM permission_groups ORDER BY name`)
		data.Groups, err = collect(rows, err, func(row *sql.Rows) (PermissionGroup, error) {
			group := PermissionGroup{Permissions: []string{}}
			err := row.Scan(&group.ID, &group.Name, &group.Description)
			return group, err
		})
		if err != nil {
			return err
		}
		groups := make(map[int64]*PermissionGroup, len(data.Groups))
		for i := range data.Groups {
			groups[data.Groups[i].ID] = &data.Groups[i]
		}
		if err := eachRow(ctx, tx, `SELECT group_id, permission FROM group_permissions ORDER BY permission`, func(row *sql.Rows) error {
			var groupID int64
			var permission string
			if err := row.Scan(&groupID, &permission); err != nil {
				return err
			}
			groups[groupID].Permissions = append(groups[groupID].Permissions, permission)
			return nil
		}); err != nil {
			return err
		}

		rows, err = tx.QueryContext(ctx, `SELECT id, name, kind, description FROM teams ORDER BY name`)
		data.Teams, err = collect(rows, err, func(row *sql.Rows) (Team, error) {
			team := Team{MemberIDs: []int64{}, AdminIDs: []int64{}}
			err := row.Scan(&team.ID, &team.Name, &team.Kind, &team.Description)
			return team, err
		})
		if err != nil {
			return err
		}
		teams := make(map[int64]*Team, len(data.Teams))
		for i := range data.Teams {
			teams[data.Teams[i].ID] = &data.Teams[i]
		}
		if err := eachRow(ctx, tx, `SELECT team_id, account_id, is_admin FROM team_members ORDER BY account_id`, func(row *sql.Rows) error {
			var teamID, accountID int64
			var admin bool
			if err := row.Scan(&teamID, &accountID, &admin); err != nil {
				return err
			}
			team := teams[teamID]
			team.MemberIDs = append(team.MemberIDs, accountID)
			if admin {
				team.AdminIDs = append(team.AdminIDs, accountID)
			}
			return nil
		}); err != nil {
			return err
		}

		return eachRow(ctx, tx, `SELECT module, account_id FROM module_admins ORDER BY account_id`, func(row *sql.Rows) error {
			var module string
			var accountID int64
			if err := row.Scan(&module, &accountID); err != nil {
				return err
			}
			for i := range data.Modules {
				if data.Modules[i].Key == module {
					data.Modules[i].AdminIDs = append(data.Modules[i].AdminIDs, accountID)
				}
			}
			return nil
		})
	})
	return data, err
}

func (s *Store) SaveAccount(ctx context.Context, id, revision int64, input AccountInput) error {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	name := strings.TrimSpace(input.Name)
	if !validName(name) || (id == 0 && !usernamePattern.MatchString(username)) || !validIDs(input.GroupIDs) {
		return ErrInvalid
	}
	if (id == 0 || input.Password != "") && !validPassword(input.Password) {
		return ErrInvalid
	}
	var hash string
	if input.Password != "" {
		var err error
		hash, err = s.newPassword(ctx, input.Password)
		if err != nil {
			return err
		}
	}
	return s.mutate(ctx, revision, func(tx *sql.Tx) error {
		if id == 0 {
			err := tx.QueryRowContext(ctx, `INSERT INTO accounts(username, name, password_hash, active)
				VALUES (?, ?, ?, ?) RETURNING id`, username, name, hash, input.Active).Scan(&id)
			if err != nil {
				return err
			}
		} else {
			var account Account
			err := account.scan(tx.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM accounts WHERE id = ?`, id))
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			if account.SuperAdmin {
				return ErrForbidden
			}
			if _, err := tx.ExecContext(ctx, `UPDATE accounts SET name = ?, active = ? WHERE id = ?`, name, input.Active, id); err != nil {
				return err
			}
			if hash != "" {
				if _, err := tx.ExecContext(ctx, `UPDATE accounts SET password_hash = ?, must_change_password = 1 WHERE id = ?`, hash, id); err != nil {
					return err
				}
			}
			if hash != "" || !input.Active {
				if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = ?`, id); err != nil {
					return err
				}
			}
		}
		return replaceLinks(ctx, tx, `DELETE FROM account_groups WHERE account_id = ?`,
			`INSERT INTO account_groups(account_id, group_id) VALUES (?, ?)`, id, input.GroupIDs)
	})
}

func (s *Store) SaveGroup(ctx context.Context, id, revision int64, group PermissionGroup) error {
	group.Name = strings.TrimSpace(group.Name)
	group.Description = strings.TrimSpace(group.Description)
	if !validName(group.Name) || utf8.RuneCountInString(group.Description) > 300 || !validPermissions(group.Permissions) {
		return ErrInvalid
	}
	return s.mutate(ctx, revision, func(tx *sql.Tx) error {
		if id == 0 {
			if err := tx.QueryRowContext(ctx, `INSERT INTO permission_groups(name, description) VALUES (?, ?) RETURNING id`, group.Name, group.Description).Scan(&id); err != nil {
				return err
			}
		} else if err := changed(tx.ExecContext(ctx, `UPDATE permission_groups SET name = ?, description = ? WHERE id = ?`, group.Name, group.Description, id)); err != nil {
			return err
		}
		return replaceLinks(ctx, tx, `DELETE FROM group_permissions WHERE group_id = ?`,
			`INSERT INTO group_permissions(group_id, permission) VALUES (?, ?)`, id, group.Permissions)
	})
}

func (s *Store) DeleteGroup(ctx context.Context, id, revision int64) error {
	return s.mutate(ctx, revision, func(tx *sql.Tx) error {
		return changed(tx.ExecContext(ctx, `DELETE FROM permission_groups WHERE id = ?`, id))
	})
}

func (s *Store) SaveTeam(ctx context.Context, id, revision int64, team Team) error {
	team.Name = strings.TrimSpace(team.Name)
	team.Description = strings.TrimSpace(team.Description)
	if !validName(team.Name) || utf8.RuneCountInString(team.Description) > 300 ||
		(team.Kind != "team" && team.Kind != "project") || !validIDs(team.MemberIDs) || !validIDs(team.AdminIDs) {
		return ErrInvalid
	}
	members := make(map[int64]bool, len(team.MemberIDs))
	for _, member := range team.MemberIDs {
		members[member] = false
	}
	for _, admin := range team.AdminIDs {
		if _, member := members[admin]; !member {
			return ErrInvalid
		}
		members[admin] = true
	}
	return s.mutate(ctx, revision, func(tx *sql.Tx) error {
		if id == 0 {
			if err := tx.QueryRowContext(ctx, `INSERT INTO teams(name, kind, description) VALUES (?, ?, ?) RETURNING id`, team.Name, team.Kind, team.Description).Scan(&id); err != nil {
				return err
			}
		} else if err := changed(tx.ExecContext(ctx, `UPDATE teams SET name = ?, kind = ?, description = ? WHERE id = ?`, team.Name, team.Kind, team.Description, id)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = ?`, id); err != nil {
			return err
		}
		for _, member := range team.MemberIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO team_members(team_id, account_id, is_admin) VALUES (?, ?, ?)`, id, member, members[member]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) DeleteTeam(ctx context.Context, id, revision int64) error {
	return s.mutate(ctx, revision, func(tx *sql.Tx) error {
		return changed(tx.ExecContext(ctx, `DELETE FROM teams WHERE id = ?`, id))
	})
}

func (s *Store) SaveModuleAdmins(ctx context.Context, key string, revision int64, ids []int64) error {
	if !validIDs(ids) {
		return ErrInvalid
	}
	for _, module := range modules {
		if module.Key != key {
			continue
		}
		return s.mutate(ctx, revision, func(tx *sql.Tx) error {
			return replaceLinks(ctx, tx, `DELETE FROM module_admins WHERE module = ?`,
				`INSERT INTO module_admins(module, account_id) VALUES (?, ?)`, key, ids)
		})
	}
	return ErrNotFound
}

func (s *Store) mutate(ctx context.Context, revision int64, fn func(*sql.Tx) error) error {
	return s.transaction(ctx, func(tx *sql.Tx) error {
		err := changed(tx.ExecContext(ctx, `UPDATE administration_revision SET revision = revision + 1 WHERE id = 1 AND revision = ?`, revision))
		if err == ErrNotFound {
			return ErrStale
		}
		if err != nil {
			return err
		}
		return fn(tx)
	})
}

func validName(name string) bool { return name != "" && utf8.RuneCountInString(name) <= 80 }

func validIDs(ids []int64) bool {
	if len(ids) > 500 {
		return false
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func validPermissions(permissions []string) bool {
	allowed := make(map[string]bool, len(modules))
	for _, module := range modules {
		allowed[module.Permission] = true
	}
	for _, permission := range permissions {
		if !allowed[permission] {
			return false
		}
		delete(allowed, permission)
	}
	return true
}

func replaceLinks[T any](ctx context.Context, tx *sql.Tx, remove, insert string, id any, links []T) error {
	if _, err := tx.ExecContext(ctx, remove, id); err != nil {
		return err
	}
	for _, link := range links {
		if _, err := tx.ExecContext(ctx, insert, id, link); err != nil {
			return err
		}
	}
	return nil
}

func eachRow(ctx context.Context, tx *sql.Tx, query string, scan func(*sql.Rows) error) error {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func scanString(row *sql.Rows) (string, error) {
	var value string
	err := row.Scan(&value)
	return value, err
}
