package goautowp

import (
	"database/sql"
	"github.com/autowp/goautowp/util"
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
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return 0, err
	}
	if err == sql.ErrNoRows {
		return 0, nil
	}

	defer util.Close(rows)

	result := 0
	if rows.Next() {
		err = rows.Scan(&result)
		if err != nil {
			return 0, err
		}
	}

	return result, nil
}

func (s *Messages) GetUserNewMessagesCount(userID int) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND NOT readen
	`, userID)
}

func (s *Messages) GetInboxCount(userID int) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id AND NOT deleted_by_to
	`, userID)
}

func (s *Messages) GetInboxNewCount(userID int) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id AND NOT deleted_by_to AND NOT readen
	`, userID)
}

func (s *Messages) GetSentCount(userID int) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE from_user_id = ? AND NOT deleted_by_from
	`, userID)
}

func (s *Messages) GetSystemCount(userID int) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id IS NULL AND NOT deleted_by_to
	`, userID)
}

func (s *Messages) GetSystemNewCount(userID int) (int, error) {
	return s.fetchCount(`
		SELECT count(1)
		FROM personal_messages
		WHERE to_user_id = ? AND from_user_id IS NULL AND NOT deleted_by_to AND NOT readen
	`, userID)
}
