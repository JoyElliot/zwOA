package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const Permission = "chat.access"
const messagePage = 50

var (
	ErrInvalid  = errors.New("聊天内容或成员信息无效")
	ErrMissing  = errors.New("找不到此会话或消息")
	ErrConflict = errors.New("此请求已提交")
	requestKey  = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

type Contact struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type Conversation struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Group       bool   `json:"group"`
	MemberCount int    `json:"memberCount"`
	LastText    string `json:"lastText"`
	LastSender  string `json:"lastSender"`
	UpdatedAt   int64  `json:"updatedAt" dc:"Unix 毫秒时间戳"`
	Unread      int    `json:"unread"`
}

type CreateInput struct {
	Title      string  `json:"title"`
	MemberIDs  []int64 `json:"memberIds" v:"required"`
	RequestKey string  `json:"requestKey" v:"required" pattern:"^[a-f0-9]{32}$" dc:"客户端生成的 32 位小写十六进制请求键"`
}

type Message struct {
	ID       int64  `json:"id"`
	SenderID int64  `json:"senderId"`
	Sender   string `json:"sender"`
	Text     string `json:"text"`
	SentAt   int64  `json:"sentAt" dc:"Unix 毫秒时间戳"`
}

type MessagePage struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

// 可搜索的同事与可加入会话的账户使用相同的权限条件。
const eligibleContacts = `a.active = 1 AND (
	a.super_admin = 1 OR EXISTS (SELECT 1 FROM module_admins ma WHERE ma.account_id = a.id AND ma.module = 'chat')
	OR EXISTS (SELECT 1 FROM account_groups ag JOIN group_permissions gp USING(group_id)
		WHERE ag.account_id = a.id AND gp.permission = '` + Permission + `'))`

func (s *Store) CanAccess(ctx context.Context, actor int64) (bool, error) {
	var allowed bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts a
		WHERE a.id = ? AND a.must_change_password = 0 AND `+eligibleContacts+`)`, actor).Scan(&allowed)
	return allowed, err
}

func (s *Store) Contacts(ctx context.Context, actor int64, query string) ([]Contact, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > 80 {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id, a.name, a.username FROM accounts a
		WHERE a.id <> ? AND `+eligibleContacts+`
		AND (instr(a.name, ?) > 0 OR instr(a.username, lower(?)) > 0 OR ? = '')
		ORDER BY a.name, a.id LIMIT 50`, actor, query, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	contacts := []Contact{}
	for rows.Next() {
		var contact Contact
		if err := rows.Scan(&contact.ID, &contact.Name, &contact.Username); err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	return contacts, rows.Err()
}

func (s *Store) Conversations(ctx context.Context, actor int64) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,
		CASE WHEN c.direct_key IS NULL THEN c.title ELSE
			(SELECT a.name FROM chat_members other JOIN accounts a ON a.id = other.account_id
			 WHERE other.conversation_id = c.id AND other.account_id <> m.account_id) END,
		c.direct_key IS NULL, (SELECT COUNT(*) FROM chat_members WHERE conversation_id = c.id),
		COALESCE(last.text, ''), COALESCE(sender.name, ''), COALESCE(last.sent_at, c.created_at),
		(SELECT COUNT(*) FROM chat_messages WHERE conversation_id = c.id AND id > m.last_read_id AND sender_id <> m.account_id)
		FROM chat_members m JOIN chat_conversations c ON c.id = m.conversation_id
		LEFT JOIN chat_messages last ON last.id = c.last_message_id AND last.conversation_id = c.id
		LEFT JOIN accounts sender ON sender.id = last.sender_id
		WHERE m.account_id = ? ORDER BY COALESCE(last.sent_at, c.created_at) DESC, c.last_message_id DESC, c.id DESC`, actor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conversations := []Conversation{}
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.Group, &c.MemberCount, &c.LastText, &c.LastSender, &c.UpdatedAt, &c.Unread); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

func (s *Store) Create(ctx context.Context, actor int64, input CreateInput) (int64, error) {
	title := strings.TrimSpace(input.Title)
	if !requestKey.MatchString(input.RequestKey) || len(input.MemberIDs) == 0 || len(input.MemberIDs) > 99 ||
		utf8.RuneCountInString(title) > 80 || (title == "" && len(input.MemberIDs) != 1) {
		return 0, ErrInvalid
	}
	seen := map[int64]bool{actor: true}
	for _, id := range input.MemberIDs {
		if id <= 0 || seen[id] {
			return 0, ErrInvalid
		}
		seen[id] = true
	}
	var directKey any
	if title == "" {
		peer := input.MemberIDs[0]
		directKey = fmt.Sprintf("%d:%d", min(actor, peer), max(actor, peer))
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM chat_conversations WHERE creator_id = ? AND request_key = ?`, actor, input.RequestKey).Scan(&id)
	if err == nil {
		return id, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	for _, member := range input.MemberIDs {
		var eligible bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts a WHERE a.id = ? AND `+eligibleContacts+`)`, member).Scan(&eligible); err != nil {
			return 0, err
		}
		if !eligible {
			return 0, ErrInvalid
		}
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO chat_conversations(title, direct_key, creator_id, request_key, created_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT(direct_key) DO NOTHING RETURNING id`, title, directKey, actor, input.RequestKey, time.Now().UnixMilli()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, `SELECT id FROM chat_conversations WHERE direct_key = ?`, directKey).Scan(&id); err != nil {
			return 0, err
		}
		return id, tx.Commit()
	}
	if err != nil {
		return 0, err
	}
	for member := range seen {
		if _, err := tx.ExecContext(ctx, `INSERT INTO chat_members(conversation_id, account_id) VALUES (?, ?)`, id, member); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
