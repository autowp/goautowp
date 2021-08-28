package goautowp

import (
	"database/sql"
	"fmt"
)

// ContactsRepository Main Object
type ContactsRepository struct {
	autowpDB *sql.DB
}

// NewContactsRepository constructor
func NewContactsRepository(db *sql.DB) (*ContactsRepository, error) {

	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	s := &ContactsRepository{
		autowpDB: db,
	}

	return s, nil
}

func (s *ContactsRepository) isExists(id int64, contactID int64) (bool, error) {
	v := 0
	err := s.autowpDB.QueryRow("SELECT 1 FROM contact WHERE user_id = ? and contact_user_id = ?", id, contactID).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *ContactsRepository) create(id int64, contactID int64) error {
	_, err := s.autowpDB.Exec(`
		INSERT IGNORE INTO contact (user_id, contact_user_id, timestamp)
		VALUES (?, ?, NOW())
    `, id, contactID)
	if err != nil {
		return err
	}

	return nil
}

func (s *ContactsRepository) delete(id int64, contactID int64) error {
	_, err := s.autowpDB.Exec("DELETE FROM contact WHERE user_id = ? AND contact_user_id = ?", id, contactID)
	if err != nil {
		return err
	}

	return nil
}
