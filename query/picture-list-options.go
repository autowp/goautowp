package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	PictureAlias = "p"
)

func AppendPictureAlias(alias string) string {
	return alias + "_" + PictureAlias
}

type PictureListOptions struct {
	Status        schema.PictureStatus
	OwnerID       int64
	PictureItem   *PictureItemListOptions
	ID            int64
	HasCopyrights bool
	OrderExpr     []exp.OrderedExpression
	Limit         uint32
	Page          uint32
}

func (s *PictureListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	sqSelect := db.Select().From(schema.PictureTable.As(PictureAlias))

	return s.Apply(PictureAlias, sqSelect)
}

func (s *PictureListOptions) CountSelect(db *goqu.Database) *goqu.SelectDataset {
	return s.Select(db).Select(
		goqu.COUNT(goqu.DISTINCT(goqu.T(PictureAlias).Col(schema.PictureTableIDColName))),
	)
}

func (s *PictureListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

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

		sqSelect = s.PictureItem.Apply(piAlias, sqSelect)
	}

	if s.HasCopyrights {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.PictureTableCopyrightsTextIDColName).IsNotNull())
	}

	if len(s.OrderExpr) > 0 {
		sqSelect = sqSelect.Order(s.OrderExpr...)
	}

	return sqSelect
}
