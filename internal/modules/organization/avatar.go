package organization

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image/jpeg"
)

const (
	maxAvatarBytes = 128 << 10
	maxAvatarSide  = 256
)

var ErrAvatar = errors.New("头像必须是有效的 JPEG，且不超过 256 × 256 像素和 128 KiB")

func (s *Store) SaveAvatar(ctx context.Context, accountID int64, encoded string) error {
	if len(encoded) == 0 || len(encoded) > base64.StdEncoding.EncodedLen(maxAvatarBytes) {
		return ErrAvatar
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) > maxAvatarBytes {
		return ErrAvatar
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > maxAvatarSide || config.Height > maxAvatarSide {
		return ErrAvatar
	}
	picture, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return ErrAvatar
	}
	var normalized bytes.Buffer
	if err := jpeg.Encode(&normalized, picture, &jpeg.Options{Quality: 86}); err != nil {
		return err
	}

	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO account_avatars(account_id, version, jpeg)
		SELECT id, ?, ? FROM accounts WHERE id = ? AND active = 1 AND must_change_password = 0
		ON CONFLICT(account_id) DO UPDATE SET version = excluded.version, jpeg = excluded.jpeg`,
		hex.EncodeToString(id[:]), normalized.Bytes(), accountID)
	err = changed(result, err)
	if errors.Is(err, ErrNotFound) {
		return ErrForbidden
	}
	return err
}

func (s *Store) Avatar(ctx context.Context, version string) (string, error) {
	if len(version) != 32 {
		return "", ErrNotFound
	}
	if _, err := hex.DecodeString(version); err != nil {
		return "", ErrNotFound
	}
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT jpeg FROM account_avatars WHERE version = ?`, version).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
