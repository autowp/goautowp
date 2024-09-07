package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	itemParentCacheAlias = "ipc"
)

func AppendItemParentCacheAlias(alias string, suffix string) string {
	return alias + "_" + itemParentCacheAlias + suffix
}

type ItemParentCacheListOptions struct {
	ItemsByParentID                 *ItemsListOptions
	ItemID                          int64
	ParentID                        int64
	ParentIDExpr                    exp.Expression
	ItemsByItemID                   *ItemsListOptions
	ItemParentByItemID              *ItemParentListOptions
	PictureItemsByItemID            *PictureItemListOptions
	ItemParentCacheAncestorByItemID *ItemParentCacheListOptions
	ExcludeSelf                     bool
}

func (s *ItemParentCacheListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	sqSelect := db.Select().From(schema.ItemParentCacheTable.As(itemParentCacheAlias))

	return s.Apply(itemParentCacheAlias, sqSelect)
}

func (s *ItemParentCacheListOptions) CountSelect(db *goqu.Database) *goqu.SelectDataset {
	return s.Select(db).Select(goqu.COUNT(goqu.Star()))
}

func (s *ItemParentCacheListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.ParentID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentCacheTableParentIDColName).Eq(s.ParentID))
	}

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentCacheTableItemIDColName).Eq(s.ItemID))
	}

	if s.ParentIDExpr != nil {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentCacheTableParentIDColName).Eq(s.ParentIDExpr))
	}

	if s.ItemsByItemID != nil {
		iAlias := AppendItemAlias(alias, "d")
		sqSelect = sqSelect.
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(aliasTable.Col(schema.ItemParentCacheTableItemIDColName).Eq(goqu.T(iAlias).Col(schema.ItemTableIDColName))),
			)

		sqSelect = s.ItemsByItemID.Apply(iAlias, sqSelect)
	}

	if s.ItemsByParentID != nil {
		iAlias := AppendItemAlias(alias, "a")
		sqSelect = sqSelect.
			Join(
				schema.ItemTable.As(iAlias),
				goqu.On(aliasTable.Col(schema.ItemParentCacheTableParentIDColName).Eq(
					goqu.T(iAlias).Col(schema.ItemTableIDColName),
				)),
			)

		sqSelect = s.ItemsByParentID.Apply(iAlias, sqSelect)
	}

	if s.ItemParentByItemID != nil {
		ippAlias := AppendItemParentAlias(alias, "p")
		sqSelect = sqSelect.Join(
			schema.ItemParentTable.As(ippAlias),
			goqu.On(aliasTable.Col(schema.ItemParentCacheTableItemIDColName).Eq(
				goqu.T(ippAlias).Col(schema.ItemParentTableItemIDColName),
			)),
		)

		sqSelect = s.ItemParentByItemID.Apply(ippAlias, sqSelect)
	}

	if s.PictureItemsByItemID != nil {
		piAlias := AppendPictureItemAlias(alias)

		sqSelect = sqSelect.Join(
			schema.PictureItemTable.As(piAlias),
			goqu.On(aliasTable.Col(schema.ItemParentCacheTableItemIDColName).Eq(
				goqu.T(piAlias).Col(schema.PictureItemTableItemIDColName),
			)),
		)

		sqSelect = s.PictureItemsByItemID.Apply(piAlias, sqSelect)
	}

	if s.ItemParentCacheAncestorByItemID != nil {
		ipcdAlias := AppendItemParentCacheAlias(alias, "d")
		sqSelect = sqSelect.
			Join(
				schema.ItemParentCacheTable.As(ipcdAlias),
				goqu.On(aliasTable.Col(schema.ItemParentCacheTableItemIDColName).Eq(
					goqu.T(ipcdAlias).Col(schema.ItemParentCacheTableItemIDColName),
				)),
			)

		sqSelect = s.ItemParentCacheAncestorByItemID.Apply(ipcdAlias, sqSelect)
	}

	if s.ExcludeSelf {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.ItemParentCacheTableItemIDColName).Neq(
			aliasTable.Col(schema.ItemParentCacheTableParentIDColName),
		))
	}

	return sqSelect
}
