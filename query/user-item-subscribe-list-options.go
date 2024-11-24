package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const (
	userItemSubscribeAlias = "uis"
)

type UserItemSubscribeListOptions struct {
	ItemIDs []int64
}

func (s *UserItemSubscribeListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	if len(s.ItemIDs) > 0 {
		sqSelect = sqSelect.Where(goqu.T(alias).Col(schema.UserItemSubscribeTableItemIDColName).In(s.ItemIDs))
	}

	return sqSelect
}
