package organization

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

const SessionLifetime = 12 * time.Hour

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,39}$`)

type Account struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Name               string `json:"name"`
	SuperAdmin         bool   `json:"superAdmin"`
	Active             bool   `json:"active"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

type Identity struct {
	Account
	AvatarVersion string           `json:"avatarVersion" dc:"当前头像版本；空字符串表示默认头像"`
	Permissions   []string         `json:"permissions"`
	Teams         []TeamMembership `json:"teams"`
	Modules       []string         `json:"adminModules"`
}

type TeamMembership struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Admin bool   `json:"admin"`
}

type scanner interface{ Scan(...any) error }

const accountColumns = `id, username, name, super_admin, active, must_change_password`

func (a *Account) scan(row scanner, extra ...any) error {
	return row.Scan(append([]any{&a.ID, &a.Username, &a.Name, &a.SuperAdmin, &a.Active, &a.MustChangePassword}, extra...)...)
}

func (s *Store) Bootstrap(ctx context.Context, username, name, password string) error {
	username = strings.ToLower(strings.TrimSpace(username))
	name = strings.TrimSpace(name)
	if !usernamePattern.MatchString(username) || !validName(name) || !validPassword(password) {
		return ErrInvalid
	}
	hash, err := s.newPassword(ctx, password)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		return changed(tx.ExecContext(ctx, `INSERT INTO accounts(username, name, password_hash, super_admin, must_change_password)
			SELECT ?, ?, ?, 1, 0 WHERE NOT EXISTS (SELECT 1 FROM accounts)`, username, name, hash))
	})
}

func (s *Store) Login(ctx context.Context, username, password, peer, previousToken string) (string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(username) || len(password) == 0 || len(password) > 256 {
		return "", ErrCredentials
	}
	if !s.attempts.allow(peer, username, time.Now()) {
		return "", ErrTooMany
	}
	var account Account
	var hash string
	err := account.scan(s.db.QueryRowContext(ctx, `SELECT `+accountColumns+`, password_hash FROM accounts WHERE username = ?`, username), &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	valid, err := s.checkPassword(ctx, password, hash)
	if err != nil {
		return "", err
	}
	if !valid || !account.Active {
		return "", ErrCredentials
	}
	token := base64.RawURLEncoding.EncodeToString(randBytes(32))
	now := time.Now()
	err = s.transaction(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now.Unix()); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO sessions(token_hash, account_id, expires_at)
			SELECT ?, id, ? FROM accounts WHERE id = ? AND active = 1 AND password_hash = ?`, tokenHash(token), now.Add(SessionLifetime).Unix(), account.ID, hash)
		err = changed(result, err)
		if errors.Is(err, ErrNotFound) {
			return ErrCredentials
		}
		if err != nil || previousToken == "" {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash(previousToken))
		return err
	})
	return token, err
}

func (s *Store) Session(ctx context.Context, token string) (Account, error) {
	var account Account
	if len(token) != 43 {
		return account, ErrUnauthorized
	}
	err := account.scan(s.db.QueryRowContext(ctx, `SELECT `+accountColumns+`
		FROM accounts WHERE active = 1 AND id = (SELECT account_id FROM sessions WHERE token_hash = ? AND expires_at > ?)`, tokenHash(token), time.Now().Unix()))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrUnauthorized
	}
	return account, err
}

func (s *Store) Identity(ctx context.Context, account Account) (Identity, error) {
	identity := Identity{Account: account, Permissions: []string{}, Teams: []TeamMembership{}, Modules: []string{}}
	if account.MustChangePassword {
		return identity, nil
	}
	if err := s.transaction(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT version FROM account_avatars WHERE account_id = ?), '')`, account.ID).
			Scan(&identity.AvatarVersion); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT DISTINCT p.permission FROM group_permissions p
			JOIN account_groups g ON g.group_id = p.group_id WHERE g.account_id = ? ORDER BY p.permission`, account.ID)
		identity.Permissions, err = collect(rows, err, scanString)
		if err != nil {
			return err
		}
		rows, err = tx.QueryContext(ctx, `SELECT t.id, t.name, t.kind, m.is_admin FROM team_members m
			JOIN teams t ON t.id = m.team_id WHERE m.account_id = ? ORDER BY t.name`, account.ID)
		identity.Teams, err = collect(rows, err, func(row *sql.Rows) (TeamMembership, error) {
			var team TeamMembership
			err := row.Scan(&team.ID, &team.Name, &team.Kind, &team.Admin)
			return team, err
		})
		if err != nil {
			return err
		}
		rows, err = tx.QueryContext(ctx, `SELECT module FROM module_admins WHERE account_id = ? ORDER BY module`, account.ID)
		identity.Modules, err = collect(rows, err, scanString)
		return err
	}); err != nil {
		return Identity{}, err
	}
	return identity, nil
}

func (s *Store) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash(token))
	return err
}

func (s *Store) ChangePassword(ctx context.Context, accountID int64, current, next string) error {
	if len(current) == 0 || len(current) > 256 || !validPassword(next) || current == next {
		return ErrInvalid
	}
	if !s.attempts.allowPasswordChange(accountID, time.Now()) {
		return ErrTooMany
	}
	var oldHash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM accounts WHERE id = ? AND active = 1`, accountID).Scan(&oldHash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnauthorized
	}
	if err != nil {
		return err
	}
	valid, err := s.checkPassword(ctx, current, oldHash)
	if err != nil {
		return err
	}
	if !valid {
		return ErrPassword
	}
	hash, err := s.newPassword(ctx, next)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		if err := changed(tx.ExecContext(ctx, `UPDATE accounts SET password_hash = ?, must_change_password = 0
			WHERE id = ? AND active = 1 AND password_hash = ?`, hash, accountID, oldHash)); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = ?`, accountID)
		return err
	})
}

func validPassword(value string) bool { return len(value) <= 256 && len([]rune(value)) >= 12 }

func randBytes(size int) []byte {
	value := make([]byte, size)
	rand.Read(value)
	return value
}

func tokenHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func (s *Store) derive(ctx context.Context, password string, salt []byte) ([]byte, error) {
	select {
	case s.hashSlots <- struct{}{}:
		defer func() { <-s.hashSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, ErrTooMany
	}
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32), nil
}

func (s *Store) newPassword(ctx context.Context, password string) (string, error) {
	salt := randBytes(16)
	key, err := s.derive(ctx, password, salt)
	if err != nil {
		return "", err
	}
	return "argon2id-v1$" + base64.RawStdEncoding.EncodeToString(append(salt, key...)), nil
}

func (s *Store) checkPassword(ctx context.Context, password, stored string) (bool, error) {
	// 未知账户也执行同样的密码计算。
	value, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, "argon2id-v1$"))
	valid := err == nil && len(value) == 48 && strings.HasPrefix(stored, "argon2id-v1$")
	if !valid {
		value = make([]byte, 48)
	}
	key, err := s.derive(ctx, password, value[:16])
	return err == nil && subtle.ConstantTimeCompare(key, value[16:]) == 1 && valid, err
}

type attemptWindow struct {
	count int
	until time.Time
}

type loginAttempts struct {
	mu    sync.Mutex
	byKey map[string]attemptWindow
	sweep time.Time
}

func (l *loginAttempts) allow(peer, username string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.expire(now)
	peerKey, accountKey := "peer:"+peer, "account:"+peer+"/"+username
	peerWindow := l.byKey[peerKey]
	if now.Before(peerWindow.until) && peerWindow.count >= 90 {
		return false
	}
	return l.consume(accountKey, 12, now) && l.consume(peerKey, 90, now)
}

func (l *loginAttempts) allowPasswordChange(accountID int64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.expire(now)
	return l.consume("change:"+strconv.FormatInt(accountID, 10), 5, now)
}

func (l *loginAttempts) expire(now time.Time) {
	if l.byKey == nil {
		l.byKey = make(map[string]attemptWindow)
	}
	if !now.Before(l.sweep) {
		for key, value := range l.byKey {
			if !now.Before(value.until) {
				delete(l.byKey, key)
			}
		}
		l.sweep = now.Add(time.Minute)
	}
}

func (l *loginAttempts) consume(key string, limit int, now time.Time) bool {
	window, exists := l.byKey[key]
	if !exists && len(l.byKey) >= 10000 {
		return false
	}
	if !now.Before(window.until) {
		window = attemptWindow{until: now.Add(time.Minute)}
	}
	if window.count >= limit {
		return false
	}
	window.count++
	l.byKey[key] = window
	return true
}
