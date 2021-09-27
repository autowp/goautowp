package goautowp

import (
	"database/sql"
)

// Messages Main Object
type Messages struct {
	db *sql.DB
}

func NewMessages(db *sql.DB) *Messages {
	return &Messages{
		db: db,
	}
}

func (s *Messages) fetchCount(query string, args ...interface{}) (int, error) {
	result := 0
	err := s.db.QueryRow(query, args...).Scan(&result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (s *Messages) GetUserNewMessagesCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND NOT readen
	`, userID)
}

func (s *Messages) GetInboxCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id AND NOT deleted_by_to
	`, userID)
}

func (s *Messages) GetInboxNewCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id AND NOT deleted_by_to AND NOT readen
	`, userID)
}

func (s *Messages) GetSentCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE from_user_id = ? AND NOT deleted_by_from
	`, userID)
}

func (s *Messages) GetSystemCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id IS NULL AND NOT deleted_by_to
	`, userID)
}

func (s *Messages) GetSystemNewCount(userID int64) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id IS NULL AND NOT deleted_by_to AND NOT readen
	`, userID)
}
