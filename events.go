package goautowp

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

type Event struct {
	UserID  int64
	Message string
	Users   []int64
}

type Events struct {
	db *goqu.Database
}

func NewEvents(db *goqu.Database) *Events {
	return &Events{
		db: db,
	}
}

func (s *Events) Add(event Event) error {
	res, err := s.db.Insert(schema.LogEventsTable).
		Rows(goqu.Record{
			schema.LogEventsTableDescriptionColName: event.Message,
			schema.LogEventsTableUserIDColName:      event.UserID,
			schema.LogEventsTableAddDatetimeColName: goqu.Func("NOW"),
		}).
		Executor().Exec()
	if err != nil {
		return err
	}

	rowID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	for _, id := range event.Users {
		_, err = s.db.Insert(schema.LogEventsUserTableName).
			Rows(goqu.Record{
				schema.LogEventsUserTableLogEventIDColName: rowID,
				schema.LogEventsUserTableNameUserIDColName: id,
			}).
			Executor().Exec()
		if err != nil {
			return err
		}
	}

	return nil
}
