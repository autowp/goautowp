package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

const (
	ItemParentCacheAlias = "ipc"
)

func AppendItemParentCacheAlias(alias string, suffix string) string {
	return alias + "_" + ItemParentCacheAlias + suffix
}

type ItemParentCacheListOptions struct {
	ItemsByParentID                 *ItemListOptions
	ItemID                          int64
	ItemIDs                         []int64
	ParentID                        int64
	ParentIDExpr                    exp.Expression
	ItemsByItemID                   *ItemListOptions
	ItemParentByItemID              *ItemParentListOptions
	PictureItemsByItemID            *PictureItemListOptions
	PictureItemsByParentID          *PictureItemListOptions
	ItemParentCacheAncestorByItemID *ItemParentCacheListOptions
	ExcludeSelf                     bool
	StockOnly                       bool
}

func (s *ItemParentCacheListOptions) Select(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	return s.apply(
		alias,
		db.Select().From(schema.ItemParentCacheTable.As(alias)),
	)
}

func (s *ItemParentCacheListOptions) CountSelect(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	sqSelect, err := s.Select(db, alias)
	if err != nil {
		return nil, err
	}

	return sqSelect.Select(goqu.COUNT(goqu.Star())), nil
}

func (s *ItemParentCacheListOptions) JoinToParentIDAndApply(
	srcCol exp.IdentifierExpression, alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, error) {
	if s == nil {
		return sqSelect, nil
	}

	return s.apply(
		alias,
		sqSelect.Join(
			schema.ItemParentCacheTable.As(alias),
			goqu.On(srcCol.Eq(goqu.T(alias).Col(schema.ItemParentCacheTableParentIDColName))),
		),
	)
}

func (s *ItemParentCacheListOptions) JoinToItemIDAndApply(
	srcCol exp.IdentifierExpression, alias string, sqSelect *goqu.SelectDataset,
) (*goqu.SelectDataset, error) {
	if s == nil {
		return sqSelect, nil
	}

	return s.apply(
		alias,
		sqSelect.Join(
			schema.ItemParentCacheTable.As(alias),
			goqu.On(srcCol.Eq(goqu.T(alias).Col(schema.ItemParentCacheTableItemIDColName))),
		),
	)
}

func (s *ItemParentCacheListOptions) apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, error) {
	var (
		err         error
		aliasTable  = goqu.T(alias)
		itemIDCol   = aliasTable.Col(schema.ItemParentCacheTableItemIDColName)
		parentIDCol = aliasTable.Col(schema.ItemParentCacheTableParentIDColName)
	)

	if s.ParentID != 0 {
		sqSelect = sqSelect.Where(parentIDCol.Eq(s.ParentID))
	}

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(itemIDCol.Eq(s.ItemID))
	}

	if len(s.ItemIDs) > 0 {
		sqSelect = sqSelect.Where(itemIDCol.In(s.ItemIDs))
	}

	if s.ParentIDExpr != nil {
		sqSelect = sqSelect.Where(parentIDCol.Eq(s.ParentIDExpr))
	}

	sqSelect, _, err = s.ItemsByItemID.JoinToIDAndApply(itemIDCol, AppendItemAlias(alias, "d"), sqSelect)
	if err != nil {
		return nil, err
	}

	sqSelect, _, err = s.ItemsByParentID.JoinToIDAndApply(parentIDCol, AppendItemAlias(alias, "a"), sqSelect)
	if err != nil {
		return nil, err
	}

	sqSelect, _, err = s.ItemParentByItemID.JoinToItemIDAndApply(
		itemIDCol,
		AppendItemParentAlias(alias, "p"),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	sqSelect, err = s.PictureItemsByItemID.JoinToItemIDAndApply(
		itemIDCol,
		AppendPictureItemAlias(alias, "i"),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	sqSelect, err = s.PictureItemsByParentID.JoinToItemIDAndApply(
		parentIDCol,
		AppendPictureItemAlias(alias, "p"),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	sqSelect, err = s.ItemParentCacheAncestorByItemID.JoinToItemIDAndApply(
		itemIDCol,
		AppendItemParentCacheAlias(alias, "d"),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	if s.ExcludeSelf {
		sqSelect = sqSelect.Where(itemIDCol.Neq(parentIDCol))
	}

	if s.StockOnly {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.ItemParentCacheTableTuningColName).IsFalse(),
			aliasTable.Col(schema.ItemParentCacheTableSportColName).IsFalse(),
		)
	}

	return sqSelect, nil
}
