package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const AttrsValuesAlias = "av"

type AttrsValuesListOptions struct {
	ZoneID      int64
	ItemID      int64
	ChildItemID int64
	AttributeID int64
	Conflict    bool
	UserValues  *AttrsUserValuesListOptions
}

func (s *AttrsValuesListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	sqSelect := db.From(schema.AttrsValuesTable.As(AttrsValuesAlias)).
		Order(goqu.T(AttrsValuesAlias).Col(schema.AttrsValuesTableUpdateDateColName).Desc())

	return s.Apply(AttrsValuesAlias, sqSelect)
}

func (s *AttrsValuesListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	if s.ItemID != 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsValuesTableItemIDColName).Eq(s.ItemID),
		)
	}

	if s.ChildItemID > 0 {
		sqSelect = sqSelect.Join(
			schema.ItemParentTable,
			goqu.On(aliasTable.Col(schema.AttrsValuesTableItemIDColName).Eq(schema.ItemParentTableParentIDCol)),
		).Where(
			schema.ItemParentTableItemIDCol.Eq(s.ChildItemID),
		)
	}

	if s.AttributeID != 0 {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsValuesTableAttributeIDColName).Eq(s.AttributeID),
		)
	}

	if s.ZoneID != 0 {
		azaAlias := AppendAttrsZoneAttributesAlias(alias)

		sqSelect = sqSelect.Join(
			schema.AttrsZoneAttributesTable.As(azaAlias),
			goqu.On(aliasTable.Col(schema.AttrsValuesTableAttributeIDColName).Eq(
				goqu.T(azaAlias).Col(schema.AttrsZoneAttributesTableAttributeIDColName),
			)),
		).Where(goqu.T(azaAlias).Col(schema.AttrsZoneAttributesTableZoneIDColName).Eq(s.ZoneID))
	}

	if s.Conflict {
		sqSelect = sqSelect.Where(
			aliasTable.Col(schema.AttrsValuesTableConflictColName).IsTrue(),
		)
	}

	if s.UserValues != nil {
		uvAlias := AppendPictureItemAlias(alias)

		sqSelect = sqSelect.Join(schema.AttrsUserValuesTable.As(uvAlias), goqu.On(
			aliasTable.Col(schema.AttrsValuesTableAttributeIDColName).Eq(
				goqu.T(uvAlias).Col(schema.AttrsUserValuesTableAttributeIDColName),
			),
			aliasTable.Col(schema.AttrsValuesTableItemIDColName).Eq(
				goqu.T(uvAlias).Col(schema.AttrsUserValuesTableItemIDColName),
			),
		))

		sqSelect = s.UserValues.Apply(uvAlias, sqSelect)
	}

	return sqSelect
}
