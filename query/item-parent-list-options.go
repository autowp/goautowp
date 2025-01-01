package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	itemParentAlias = "ip"
)

func AppendItemParentAlias(alias string, suffix string) string {
	return alias + "_" + itemParentAlias + suffix
}

type ItemParentListOptions struct {
	ItemID                            int64
	ParentID                          int64
	ParentIDExpr                      exp.Expression
	LinkedInDays                      int
	ParentItems                       *ItemsListOptions
	ChildItems                        *ItemsListOptions
	ItemParentParentByChildID         *ItemParentListOptions
	ItemParentCacheAncestorByParentID *ItemParentCacheListOptions
	Language                          string
}

func (s *ItemParentListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	sqSelect := db.Select().From(schema.ItemParentTable.As(itemParentAlias))

	return s.Apply(itemParentAlias, sqSelect)
}

func (s *ItemParentListOptions) CountSelect(db *goqu.Database) *goqu.SelectDataset {
	return s.Select(db).Select(goqu.COUNT(goqu.Star()))
}

func (s *ItemParentListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableItemIDColName).Eq(s.ItemID))
	}

	if s.ParentID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(s.ParentID))
	}

	if s.ParentIDExpr != nil {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(s.ParentIDExpr))
	}

	if s.LinkedInDays > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableTimestampColName).Gt(
			goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL ? DAY", s.LinkedInDays)),
		))
	}

	if s.ParentItems != nil {
		iAlias := AppendItemAlias(alias, "p")

		sqSelect = sqSelect.
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(
					goqu.T(iAlias).Col(schema.ItemTableIDColName),
				)),
			)

		sqSelect = s.ParentItems.Apply(iAlias, sqSelect)
	}

	if s.ChildItems != nil {
		iAlias := AppendItemAlias(alias, "c")

		sqSelect = sqSelect.
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(aliasTable.Col(schema.ItemParentTableItemIDColName).Eq(
					goqu.T(iAlias).Col(schema.ItemTableIDColName),
				)),
			)

		sqSelect = s.ChildItems.Apply(iAlias, sqSelect)
	}

	if s.ItemParentCacheAncestorByParentID != nil {
		ipcaAlias := AppendItemParentCacheAlias(alias, "a")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcaAlias),
				goqu.On(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(
					goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableItemIDColName),
				)),
			)

		sqSelect = s.ItemParentCacheAncestorByParentID.Apply(ipcaAlias, sqSelect)
	}

	if s.ItemParentParentByChildID != nil {
		ippAlias := AppendItemParentAlias(alias, "p")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentTable.As(ippAlias),
				goqu.On(aliasTable.Col(schema.ItemParentTableItemIDColName).Eq(
					goqu.T(ippAlias).Col(schema.ItemParentTableItemIDColName),
				)),
			)

		sqSelect = s.ItemParentParentByChildID.Apply(ippAlias, sqSelect)
	}

	return sqSelect
}
