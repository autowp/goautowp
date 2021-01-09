package goautowp

import (
	"database/sql"
	"fmt"
	"github.com/autowp/goautowp/util"
)

// UserRepository Main Object
type UserRepository struct {
	autowpDB *sql.DB
}

// NewUserRepository constructor
func NewUserRepository(autowpDB *sql.DB) (*UserRepository, error) {

	if autowpDB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	s := &UserRepository{
		autowpDB: autowpDB,
	}

	return s, nil
}

func (s *UserRepository) GetUser(id int) (*DBUser, error) {
	if id == 0 {
		return nil, nil
	}

	rows, err := s.autowpDB.Query(`
		SELECT id, name, deleted, identity, last_online, role
		FROM users
		WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	if !rows.Next() {
		return nil, nil
	}

	var r DBUser
	err = rows.Scan(&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
