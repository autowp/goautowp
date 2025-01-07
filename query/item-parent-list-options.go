package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	ItemParentAlias = "ip"
)

func AppendItemParentAlias(alias string, suffix string) string {
	return alias + "_" + ItemParentAlias + suffix
}

type ItemParentListOptions struct {
	ItemID                            int64
	ParentID                          int64
	Type                              schema.ItemParentType
	ParentIDExpr                      exp.Expression
	LinkedInDays                      int
	ParentItems                       *ItemsListOptions
	ChildItems                        *ItemsListOptions
	ItemParentParentByChildID         *ItemParentListOptions
	ItemParentCacheAncestorByParentID *ItemParentCacheListOptions
	ItemParentCacheAncestorByChildID  *ItemParentCacheListOptions
	Language                          string
	Limit                             uint32
	Page                              uint32
	Catname                           string
}

func (s *ItemParentListOptions) Select(db *goqu.Database) (*goqu.SelectDataset, bool, error) {
	sqSelect := db.Select().From(schema.ItemParentTable.As(ItemParentAlias))

	return s.Apply(ItemParentAlias, sqSelect)
}

func (s *ItemParentListOptions) CountSelect(db *goqu.Database) (*goqu.SelectDataset, error) {
	sqSelect, groupBy, err := s.Select(db)
	if err != nil {
		return nil, err
	}

	if groupBy {
		sqSelect = sqSelect.Select(goqu.COUNT(goqu.DISTINCT(goqu.Star())))
	} else {
		sqSelect = sqSelect.Select(goqu.COUNT(goqu.Star()))
	}

	return sqSelect, nil
}

func (s *ItemParentListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, bool, error) {
	var (
		err        error
		groupBy    = false
		subGroupBy = false
		aliasTable = goqu.T(alias)
	)

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableItemIDColName).Eq(s.ItemID))
	}

	if s.ParentID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(s.ParentID))
	}

	if s.ParentIDExpr != nil {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(s.ParentIDExpr))
	}

	if s.Type != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableTypeColName).Eq(s.Type))
	}

	if s.LinkedInDays > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableTimestampColName).Gt(
			goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL ? DAY", s.LinkedInDays)),
		))
	}

	if s.Catname != "" {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentTableCatnameColName).Eq(s.Catname))
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

		sqSelect, subGroupBy, err = s.ParentItems.Apply(iAlias, sqSelect)
		if err != nil {
			return nil, false, err
		}

		if subGroupBy {
			groupBy = true
		}
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

		sqSelect, groupBy, err = s.ChildItems.Apply(iAlias, sqSelect)
		if err != nil {
			return nil, false, err
		}

		if subGroupBy {
			groupBy = true
		}
	}

	if s.ItemParentCacheAncestorByParentID != nil {
		ipcaAlias := AppendItemParentCacheAlias(alias, "ap")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcaAlias),
				goqu.On(aliasTable.Col(schema.ItemParentTableParentIDColName).Eq(
					goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableItemIDColName),
				)),
			)

		sqSelect, err = s.ItemParentCacheAncestorByParentID.Apply(ipcaAlias, sqSelect)
		if err != nil {
			return nil, false, err
		}

		groupBy = true
	}

	if s.ItemParentCacheAncestorByChildID != nil {
		ipcaAlias := AppendItemParentCacheAlias(alias, "ac")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcaAlias),
				goqu.On(aliasTable.Col(schema.ItemParentTableItemIDColName).Eq(
					goqu.T(ipcaAlias).Col(schema.ItemParentCacheTableItemIDColName),
				)),
			)

		sqSelect, err = s.ItemParentCacheAncestorByParentID.Apply(ipcaAlias, sqSelect)
		if err != nil {
			return nil, false, err
		}

		groupBy = true
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

		sqSelect, _, err = s.ItemParentParentByChildID.Apply(ippAlias, sqSelect)
		if err != nil {
			return nil, false, err
		}

		groupBy = true
	}

	return sqSelect, groupBy, nil
}
