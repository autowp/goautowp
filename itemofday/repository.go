package itemofday

import (
	"context"
	"github.com/doug-martin/goqu/v9"
	"time"
)

type Repository struct {
	db  *goqu.Database
	loc *time.Location
}

type NextDate struct {
	Date time.Time
	Free bool
}

func NewRepository(db *goqu.Database) *Repository {
	loc, _ := time.LoadLocation("UTC")

	return &Repository{
		db:  db,
		loc: loc,
	}
}

func (s *Repository) NextDates(ctx context.Context) ([]NextDate, error) {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, s.loc)

	result := make([]NextDate, 0)

	for i := 0; i < 10; i++ {
		found := false
		_, err := s.db.Select(goqu.L("1")).From("of_day").Where(
			goqu.I("day_date").Eq(now.Format("2006-01-02")),
			goqu.I("item_id").IsNotNull(),
		).Executor().ScanValContext(ctx, &found)

		if err != nil {
			return nil, err
		}

		result = append(result, NextDate{
			Date: now,
			Free: !found,
		})

		now = now.AddDate(0, 0, 1)
	}

	return result, nil
}
