package messaging

import (
	"context"
	"database/sql"
	"errors"
	"github.com/autowp/goautowp/telegram"
	"strings"
)

const MaxText = 2000

type Repository struct {
	db              *sql.DB
	telegramService *telegram.Service
}

func NewRepository(db *sql.DB, telegramService *telegram.Service) *Repository {
	return &Repository{
		db:              db,
		telegramService: telegramService,
	}
}

func (s *Repository) fetchCount(query string, args ...interface{}) (int, error) {
	result := 0
	err := s.db.QueryRow(query, args...).Scan(&result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (s *Repository) GetUserNewMessagesCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND NOT readen
	`, userID)
}

func (s *Repository) GetInboxCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id AND NOT deleted_by_to
	`, userID)
}

func (s *Repository) GetInboxNewCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id AND NOT deleted_by_to AND NOT readen
	`, userID)
}

func (s *Repository) GetSentCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE from_user_id = ? AND NOT deleted_by_from
	`, userID)
}

func (s *Repository) GetSystemCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id IS NULL AND NOT deleted_by_to
	`, userID)
}

func (s *Repository) GetSystemNewCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id IS NULL AND NOT deleted_by_to AND NOT readen
	`, userID)
}

func (s *Repository) DeleteMessage(ctx context.Context, userID int64, messageID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE personal_messages SET deleted_by_from = 1 WHERE from_user_id = ? AND id = ?
    `, userID, messageID)

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE personal_messages SET deleted_by_to = 1 WHERE to_user_id = ? AND id = ?
    `, userID, messageID)

	return err
}

func (s *Repository) ClearSent(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE personal_messages SET deleted_by_from = 1 WHERE from_user_id = ?
    `, userID)

	return err
}

func (s *Repository) ClearSystem(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM personal_messages WHERE to_user_id = ? AND from_user_id IS NULL
    `, userID)

	return err
}

func (s *Repository) CreateMessage(ctx context.Context, fromUserID int64, toUserID int64, message string) error {
	message = strings.TrimSpace(message)
	msgLength := len(message)

	if msgLength <= 0 {
		return errors.New("message is empty")
	}

	if msgLength > MaxText {
		return errors.New("too long message")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO personal_messages (from_user_id, to_user_id, contents, add_datetime, readen) VALUES (?, ?, ?, NOW(), 0)
    `, fromUserID, toUserID, message)
	if err != nil {
		return err
	}

	err = s.telegramService.NotifyMessage(ctx, fromUserID, toUserID, message)
	if err != nil {
		return err
	}

	return nil
}
