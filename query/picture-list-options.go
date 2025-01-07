package query

import (
	"errors"
	"time"

	"github.com/autowp/goautowp/schema"
	"github.com/autowp/goautowp/util"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

var errNoTimezone = errors.New("timezone not provided")

const (
	PictureAlias = "p"
)

func AppendPictureAlias(alias string) string {
	return alias + "_" + PictureAlias
}

type PictureListOptions struct {
	Status         schema.PictureStatus
	OwnerID        int64
	PictureItem    *PictureItemListOptions
	ID             int64
	HasCopyrights  bool
	OrderExpr      []exp.OrderedExpression
	Limit          uint32
	Page           uint32
	AcceptedInDays int32
	AddDate        *util.Date
	AcceptDate     *util.Date
	Timezone       *time.Location
}

func (s *PictureListOptions) Select(db *goqu.Database) (*goqu.SelectDataset, error) {
	sqSelect := db.Select().From(schema.PictureTable.As(PictureAlias))

	return s.Apply(PictureAlias, sqSelect)
}

func (s *PictureListOptions) CountSelect(db *goqu.Database) (*goqu.SelectDataset, error) {
	sqSelect, err := s.Select(db)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(
		goqu.COUNT(goqu.DISTINCT(goqu.T(PictureAlias).Col(schema.PictureTableIDColName))),
	), nil
}

func (s *PictureListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, error) {
	var (
		err        error
		aliasTable = goqu.T(alias)
	)

	if s.ID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableIDColName).Eq(s.ID))
	}

	if s.Status != "" {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableStatusColName).Eq(s.Status))
	}

	if s.OwnerID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableOwnerIDColName).Eq(s.OwnerID))
	}

	if s.PictureItem != nil {
		piAlias := AppendPictureItemAlias(alias)

		sqSelect = sqSelect.Join(
			schema.PictureItemTable.As(piAlias),
			goqu.On(aliasTable.Col(schema.PictureTableIDColName).Eq(
				goqu.T(piAlias).Col(schema.PictureItemTablePictureIDColName),
			)),
		)

		sqSelect, err = s.PictureItem.Apply(piAlias, sqSelect)
		if err != nil {
			return nil, err
		}
	}

	if s.HasCopyrights {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableCopyrightsTextIDColName).IsNotNull())
	}

	if s.AcceptedInDays > 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.PictureTableAcceptDatetimeColName).Gt(
				goqu.Func("DATE_SUB", goqu.Func("CURDATE"), goqu.L("INTERVAL ? DAY", s.AcceptedInDays)),
			),
		)
	}

	if s.AddDate != nil {
		sqSelect, err = s.setDateFilter(
			sqSelect, aliasTable.Col(schema.PictureTableAddDateColName), *s.AddDate, s.Timezone,
		)
		if err != nil {
			return nil, err
		}
	}

	if s.AcceptDate != nil {
		sqSelect, err = s.setDateFilter(
			sqSelect, aliasTable.Col(schema.PictureTableAcceptDatetimeColName), *s.AcceptDate, s.Timezone,
		)
		if err != nil {
			return nil, err
		}
	}

	if len(s.OrderExpr) > 0 {
		sqSelect = sqSelect.Order(s.OrderExpr...)
	}

	return sqSelect, nil
}

func (s *PictureListOptions) setDateFilter(
	sqSelect *goqu.SelectDataset, column exp.IdentifierExpression, date util.Date, timezone *time.Location,
) (*goqu.SelectDataset, error) {
	if s.Timezone == nil {
		return nil, errNoTimezone
	}

	start := time.Date(date.Year, date.Month, date.Day, 0, 0, 0, 0, timezone).In(time.UTC)
	end := time.Date(date.Year, date.Month, date.Day, 23, 59, 59, 999, timezone).In(time.UTC)

	return sqSelect.Where(
		column.Between(goqu.Range(start.Format(time.DateTime), end.Format(time.DateTime))),
	), nil
}
