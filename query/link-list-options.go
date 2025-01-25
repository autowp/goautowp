package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	LinkAlias = "l"
)

type LinkListOptions struct {
	ID                        int64
	ItemID                    int64
	Type                      string
	ItemParentCacheDescendant *ItemParentCacheListOptions
}

func (s *LinkListOptions) IsIDUnique() bool {
	return s.ItemParentCacheDescendant == nil
}

func (s *LinkListOptions) Select(db *goqu.Database, alias string) (*goqu.SelectDataset, error) {
	return s.apply(
		alias,
		db.Select().From(schema.LinksTable.As(alias)),
	)
}

func (s *LinkListOptions) apply(alias string, sqSelect *goqu.SelectDataset) (*goqu.SelectDataset, error) {
	if s == nil {
		return sqSelect, nil
	}

	var (
		aliasTable = goqu.T(alias)
		err        error
		itemIDCol  = aliasTable.Col(schema.LinksTableIDColName)
	)

	if s.ID != 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.LinksTableIDColName).Eq(s.ID))
	}

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(itemIDCol.Eq(s.ItemID))
	}

	if len(s.Type) > 0 {
		sqSelect = sqSelect.Where(aliasTable.Col(schema.LinksTableTypeColName).Eq(s.Type))
	}

	sqSelect, err = s.ItemParentCacheDescendant.JoinToParentIDAndApply(
		itemIDCol,
		AppendItemParentCacheAlias(alias, "d"),
		sqSelect,
	)
	if err != nil {
		return nil, err
	}

	return sqSelect, nil
}
