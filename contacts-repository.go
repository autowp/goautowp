package goautowp

import (
	"context"
	"database/sql"
	"errors"

	"github.com/doug-martin/goqu/v9"
)

// ContactsRepository Main Object.
type ContactsRepository struct {
	autowpDB *goqu.Database
}

// NewContactsRepository constructor.
func NewContactsRepository(db *goqu.Database) *ContactsRepository {
	return &ContactsRepository{
		autowpDB: db,
	}
}

func (s *ContactsRepository) isExists(ctx context.Context, id int64, contactID int64) (bool, error) {
	v := 0
	err := s.autowpDB.QueryRowContext(
		ctx,
		"SELECT 1 FROM contact WHERE user_id = ? and contact_user_id = ?",
		id, contactID,
	).Scan(&v)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *ContactsRepository) create(ctx context.Context, id int64, contactID int64) error {
	_, err := s.autowpDB.ExecContext(ctx, `
		INSERT IGNORE INTO contact (user_id, contact_user_id, timestamp)
		VALUES (?, ?, NOW())
    `, id, contactID)
	if err != nil {
		return err
	}

	return nil
}

func (s *ContactsRepository) delete(ctx context.Context, id int64, contactID int64) error {
	_, err := s.autowpDB.ExecContext(ctx, "DELETE FROM contact WHERE user_id = ? AND contact_user_id = ?", id, contactID)
	if err != nil {
		return err
	}

	return nil
}

func (s *ContactsRepository) deleteUserEverywhere(ctx context.Context, id int64) error {
	_, err := s.autowpDB.ExecContext(ctx, "DELETE FROM contact WHERE user_id = ?", id)
	if err != nil {
		return err
	}

	_, err = s.autowpDB.ExecContext(ctx, "DELETE FROM contact WHERE contact_user_id = ?", id)
	if err != nil {
		return err
	}

	return nil
}
