package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	itemParentAlias = "ip"
)

func AppendItemParentAlias(alias string, suffix string) string {
	return alias + "_" + itemParentAlias + suffix
}

type ItemParentListOptions struct {
	ParentID     int64
	LinkedInDays int
	ParentItems  *ItemsListOptions
	ChildItems   *ItemsListOptions
}

func (s *ItemParentListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.ParentID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(s.ParentID))
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

	return sqSelect
}
