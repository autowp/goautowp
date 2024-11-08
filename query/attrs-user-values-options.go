package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const AttrsUserValuesAlias = "auv"

type AttrsUserValuesListOptions struct {
	ZoneID         int64
	AttributeID    int64
	ItemID         int64
	UserID         int64
	ExcludeUserID  int64
	WeightLtZero   bool
	ConflictLtZero bool
	ConflictGtZero bool
}

func (s *AttrsUserValuesListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	sqSelect := db.From(schema.AttrsUserValuesTable.As(AttrsUserValuesAlias)).
		Order(goqu.T(AttrsUserValuesAlias).Col(schema.AttrsUserValuesTableUpdateDateColName).Desc())

	return s.Apply(AttrsUserValuesAlias, sqSelect)
}

func (s *AttrsUserValuesListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	sqSelect = sqSelect.Where(
		aliasTable.Col(schema.AttrsUserValuesTableItemIDColName).Eq(s.ItemID),
	)

	if s.ZoneID != 0 {
		azaAlias := AppendAttrsZoneAttributesAlias(alias)

		sqSelect = sqSelect.Join(
			schema.AttrsZoneAttributesTable.As(azaAlias),
			goqu.On(aliasTable.Col(schema.AttrsUserValuesTableAttributeIDColName).Eq(
				goqu.T(azaAlias).Col(schema.AttrsZoneAttributesTableAttributeIDColName),
			)),
		).Where(goqu.T(azaAlias).Col(schema.AttrsZoneAttributesTableZoneIDColName).Eq(s.ZoneID))
	}

	if s.AttributeID != 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsUserValuesTableAttributeIDColName).Eq(s.AttributeID),
		)
	}

	if s.UserID != 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsUserValuesTableUserIDColName).Eq(s.UserID),
		)
	}

	if s.ExcludeUserID != 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsUserValuesTableUserIDColName).Neq(s.ExcludeUserID),
		)
	}

	if s.WeightLtZero {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsUserValuesTableWeightColName).Lt(0),
		)
	}

	if s.ConflictLtZero {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsUserValuesTableConflictColName).Lt(0),
		)
	}

	if s.ConflictGtZero {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsUserValuesTableConflictColName).Gt(0),
		)
	}

	return sqSelect
}