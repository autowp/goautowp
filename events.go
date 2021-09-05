package goautowp

import (
	"database/sql"
)

type Event struct {
	UserID  int64
	Message string
	Users   []int64
}

type Events struct {
	db *sql.DB
}

func NewEvents(db *sql.DB) *Events {
	return &Events{
		db: db,
	}
}

func (s *Events) Add(event Event) error {
	r, err := s.db.Exec("INSERT INTO log_events (description, user_id, add_datetime) VALUES (?, ?, NOW())", event.Message, event.UserID)
	if err != nil {
		return err
	}

	rowId, err := r.LastInsertId()
	if err != nil {
		return err
	}

	for _, id := range event.Users {
		_, err = s.db.Exec("INSERT INTO log_events_user (log_event_id, user_id) VALUES (?, ?)", rowId, id)
		if err != nil {
			return err
		}
	}

	return nil
}
