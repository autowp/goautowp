package goautowp

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/autowp/goautowp/util"
	"time"
)

type GetUsersOptions struct {
	ID         int
	InContacts int
	Order      []string
	Fields     map[string]bool
	Deleted    *bool
}

// APIUser APIUser
type APIUser struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Deleted    bool       `json:"deleted"`
	LongAway   bool       `json:"long_away"`
	Green      bool       `json:"green"`
	Route      []string   `json:"route"`
	Identity   *string    `json:"identity"`
	Avatar     *string    `json:"avatar,omitempty"`
	Gravatar   *string    `json:"gravatar,omitempty"`
	LastOnline *time.Time `json:"last_online,omitempty"`
}

// DBUser DBUser
type DBUser struct {
	ID         int
	Name       string
	Deleted    bool
	Identity   *string
	LastOnline *time.Time
	Role       string
	EMail      *string
	Img        *int
}

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

func (s *UserRepository) GetUser(options GetUsersOptions) (*DBUser, error) {

	users, err := s.GetUsers(options)
	if err != nil {
		return nil, err
	}

	if len(users) <= 0 {
		return nil, nil
	}

	return &users[0], nil
}

func (s *UserRepository) GetUsers(options GetUsersOptions) ([]DBUser, error) {

	result := make([]DBUser, 0)

	var r DBUser
	valuePtrs := []interface{}{&r.ID, &r.Name, &r.Deleted, &r.Identity, &r.LastOnline, &r.Role}

	sqSelect := sq.Select("users.id, users.name, users.deleted, users.identity, users.last_online, users.role").From("users")

	if options.ID != 0 {
		sqSelect = sqSelect.Where(sq.Eq{"users.id": options.ID})
	}

	if options.InContacts != 0 {
		sqSelect = sqSelect.Join("contact ON users.id = contact.contact_user_id").Where(sq.Eq{"contact.user_id": options.InContacts})
	}

	if options.Deleted != nil {
		if *options.Deleted {
			sqSelect = sqSelect.Where("users.deleted")
		} else {
			sqSelect = sqSelect.Where("not users.deleted")
		}
	}

	if len(options.Order) > 0 {
		sqSelect = sqSelect.OrderBy(options.Order...)
	}

	if len(options.Fields) > 0 {
		for field := range options.Fields {
			switch field {
			case "avatar":
				sqSelect = sqSelect.Columns("users.img")
				valuePtrs = append(valuePtrs, &r.Img)
			case "gravatar":
				sqSelect = sqSelect.Columns("users.e_mail")
				valuePtrs = append(valuePtrs, &r.EMail)
			}
		}
	}

	rows, err := sqSelect.RunWith(s.autowpDB).Query()
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	defer util.Close(rows)

	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}

	return result, nil
}
