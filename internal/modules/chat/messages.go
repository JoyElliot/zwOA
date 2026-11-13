package chat

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type SendInput struct {
	Text       string `json:"text" v:"required"`
	RequestKey string `json:"requestKey" v:"required" pattern:"^[a-f0-9]{32}$" dc:"客户端生成的 32 位小写十六进制请求键"`
}

const messageColumns = `m.id, m.sender_id, a.name, m.text, m.sent_at`

type messageScanner interface{ Scan(...any) error }

func (m *Message) scan(row messageScanner) error {
	return row.Scan(&m.ID, &m.SenderID, &m.Sender, &m.Text, &m.SentAt)
}

func (s *Store) Messages(ctx context.Context, actor, conversation, before, after int64) (MessagePage, error) {
	page := MessagePage{Messages: []Message{}, Avatars: []MemberAvatar{}}
	err := s.withMember(ctx, actor, conversation, func(tx *sql.Tx) error {
		avatars, err := tx.QueryContext(ctx, `SELECT avatar.account_id, avatar.version
			FROM chat_members member JOIN account_avatars avatar ON avatar.account_id = member.account_id
			WHERE member.conversation_id = ? ORDER BY avatar.account_id`, conversation)
		if err != nil {
			return err
		}
		defer avatars.Close()
		for avatars.Next() {
			var avatar MemberAvatar
			if err := avatars.Scan(&avatar.AccountID, &avatar.Version); err != nil {
				return err
			}
			page.Avatars = append(page.Avatars, avatar)
		}
		if err := avatars.Err(); err != nil {
			return err
		}

		query := `SELECT ` + messageColumns + ` FROM chat_messages m JOIN accounts a ON a.id = m.sender_id WHERE m.conversation_id = ?`
		args := []any{conversation}
		if after > 0 {
			query += ` AND m.id > ? ORDER BY m.id ASC`
			args = append(args, after)
		} else {
			if before > 0 {
				query += ` AND m.id < ?`
				args = append(args, before)
			}
			query += ` ORDER BY m.id DESC`
		}
		rows, err := tx.QueryContext(ctx, query+` LIMIT ?`, append(args, messagePage+1)...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var message Message
			if err := message.scan(rows); err != nil {
				return err
			}
			page.Messages = append(page.Messages, message)
		}
		if len(page.Messages) > messagePage {
			page.HasMore = true
			page.Messages = page.Messages[:messagePage]
		}
		if after == 0 {
			slices.Reverse(page.Messages)
		}
		return rows.Err()
	})
	return page, err
}

func (s *Store) Send(ctx context.Context, actor, conversation int64, input SendInput) (Message, error) {
	var message Message
	input.Text = strings.TrimSpace(input.Text)
	if input.Text == "" || utf8.RuneCountInString(input.Text) > 5000 || !requestKey.MatchString(input.RequestKey) {
		return message, ErrInvalid
	}
	err := s.withMember(ctx, actor, conversation, func(tx *sql.Tx) error {
		var id int64
		err := tx.QueryRowContext(ctx, `INSERT INTO chat_messages(conversation_id, sender_id, request_key, text, sent_at)
			VALUES (?, ?, ?, ?, ?) ON CONFLICT(conversation_id, sender_id, request_key) DO NOTHING RETURNING id`,
			conversation, actor, input.RequestKey, input.Text, time.Now().UnixMilli()).Scan(&id)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if err := message.scan(tx.QueryRowContext(ctx, `SELECT `+messageColumns+`
				FROM chat_messages m JOIN accounts a ON a.id = m.sender_id
				WHERE m.conversation_id = ? AND m.sender_id = ? AND m.request_key = ?`, conversation, actor, input.RequestKey)); err != nil {
				return err
			}
			if message.Text != input.Text {
				return ErrConflict
			}
			return nil
		case err != nil:
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE chat_conversations SET last_message_id = ? WHERE id = ?`, id, conversation); err != nil {
			return err
		}
		return message.scan(tx.QueryRowContext(ctx, `SELECT `+messageColumns+`
			FROM chat_messages m JOIN accounts a ON a.id = m.sender_id WHERE m.id = ?`, id))
	})
	return message, err
}

func (s *Store) Read(ctx context.Context, actor, conversation, message int64) error {
	if message <= 0 {
		return ErrInvalid
	}
	return s.withMember(ctx, actor, conversation, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE chat_members SET last_read_id = MAX(last_read_id, ?)
			WHERE account_id = ? AND conversation_id = ?
			AND EXISTS (SELECT 1 FROM chat_messages WHERE id = ? AND conversation_id = ?)`,
			message, actor, conversation, message, conversation)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err == nil && count == 0 {
			return ErrMissing
		}
		return err
	})
}

func (s *Store) withMember(ctx context.Context, actor, conversation int64, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var allowed bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chat_members m JOIN accounts a ON a.id = m.account_id
		WHERE m.conversation_id = ? AND a.id = ? AND a.must_change_password = 0 AND `+eligibleContacts+`)`, conversation, actor).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrMissing
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
