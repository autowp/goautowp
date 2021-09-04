package auth

import (
	"database/sql"
	"fmt"
	"strings"
)

// UserStore UserStore
type UserStore struct {
	salt string
	db   *sql.DB
}

// User User
type User struct {
	ID    int64
	Login *string
	EMail *string
	Name  string
}

// NewUserStore constructor
func NewUserStore(db *sql.DB, salt string) *UserStore {
	return &UserStore{
		salt: salt,
		db:   db,
	}
}

// GetUserByCredentials GetUserByCredentials
func (s *UserStore) GetUserByCredentials(username string, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, nil
	}

	column := "login"
	if strings.Contains(username, "@") {
		column = "e_mail"
	}

	item := &User{}

	err := s.db.QueryRow(
		fmt.Sprintf(
			`
				SELECT id, login, e_mail, name
				FROM users
				WHERE NOT deleted AND %s = ? AND password = MD5(CONCAT(?, ?))
			`,
			column,
		),
		username, s.salt, password,
	).Scan(&item.ID, &item.Login, &item.EMail, &item.Name)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return item, nil
}
