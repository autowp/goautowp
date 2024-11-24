package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

type UserListOptions struct {
	ID            int64
	IDs           []int64
	ExcludeIDs    []int64
	Identity      string
	InContacts    int64
	Order         []exp.OrderedExpression
	Deleted       *bool
	HasSpecs      *bool
	IsOnline      bool
	HasPictures   *bool
	Limit         uint64
	Page          uint64
	Search        string
	ItemSubscribe *UserItemSubscribeListOptions
}

func (s *UserListOptions) Apply(sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	alias := schema.UserTableName
	aliasTable := goqu.T(alias)

	if s.ID != 0 {
		sqSelect = sqSelect.Where(schema.UserTableIDCol.Eq(s.ID))
	}

	if len(s.IDs) != 0 {
		sqSelect = sqSelect.Where(schema.UserTableIDCol.In(s.IDs))
	}

	if len(s.ExcludeIDs) != 0 {
		sqSelect = sqSelect.Where(schema.UserTableIDCol.NotIn(s.ExcludeIDs))
	}

	if len(s.Identity) > 0 {
		sqSelect = sqSelect.Where(schema.UserTableIdentityCol.Eq(s.Identity))
	}

	if len(s.Search) > 0 {
		sqSelect = sqSelect.Where(schema.UserTableNameCol.ILike(s.Search + "%"))
	}

	if s.InContacts != 0 {
		sqSelect = sqSelect.Join(
			schema.ContactTable,
			goqu.On(schema.UserTableIDCol.Eq(schema.ContactTableContactUserIDCol))).
			Where(schema.ContactTableUserIDCol.Eq(s.InContacts))
	}

	if s.Deleted != nil {
		if *s.Deleted {
			sqSelect = sqSelect.Where(schema.UserTableDeletedCol.IsTrue())
		} else {
			sqSelect = sqSelect.Where(schema.UserTableDeletedCol.IsFalse())
		}
	}

	if s.HasSpecs != nil {
		if *s.HasSpecs {
			sqSelect = sqSelect.Where(schema.UserTableSpecsVolumeCol.Gt(0))
		} else {
			sqSelect = sqSelect.Where(schema.UserTableSpecsVolumeCol.Eq(0))
		}
	}

	if s.HasPictures != nil {
		if *s.HasPictures {
			sqSelect = sqSelect.Where(schema.UserTablePicturesTotalCol.Gt(0))
		} else {
			sqSelect = sqSelect.Where(schema.UserTablePicturesTotalCol.Eq(0))
		}
	}

	if s.IsOnline {
		sqSelect = sqSelect.Where(schema.UserTableLastOnlineCol.Gte(
			goqu.Func("DATE_SUB", goqu.Func("NOW"), goqu.L("INTERVAL 5 MINUTE")),
		))
	}

	if len(s.Order) > 0 {
		sqSelect = sqSelect.Order(s.Order...)
	}

	if s.ItemSubscribe != nil {
		isAlias := alias + "_" + userItemSubscribeAlias
		sqSelect = sqSelect.Join(
			schema.UserItemSubscribeTable.As(isAlias),
			goqu.On(aliasTable.Col(schema.UserTableIDColName).Eq(
				goqu.T(isAlias).Col(schema.UserItemSubscribeTableUserIDColName),
			)),
		)

		sqSelect = s.ItemSubscribe.Apply(isAlias, sqSelect)
	}

	return sqSelect
}
