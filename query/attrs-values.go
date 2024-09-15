package query

import (
	"github.com/autowp/goautowp/schema"
	"github.com/doug-martin/goqu/v9"
)

const AttrsValuesAlias = "av"

type AttrsValuesListOptions struct {
	ZoneID int64
	ItemID int64
}

func (s *AttrsValuesListOptions) Select(db *goqu.Database) *goqu.SelectDataset {
	sqSelect := db.From(schema.AttrsValuesTable.As(AttrsValuesAlias)).
		Order(goqu.T(AttrsValuesAlias).Col(schema.AttrsValuesTableUpdateDateColName).Desc())

	return s.Apply(AttrsValuesAlias, sqSelect)
}

func (s *AttrsValuesListOptions) Apply(alias string, sqSelect *goqu.SelectDataset) *goqu.SelectDataset {
	aliasTable := goqu.T(alias)

	sqSelect = sqSelect.Where(
		aliasTable.Col(schema.AttrsValuesTableItemIDColName).Eq(s.ItemID),
	)

	if s.ZoneID != 0 {
		azaAlias := AppendAttrsZoneAttributesAlias(alias)

		sqSelect = sqSelect.Join(
			schema.AttrsZoneAttributesTable.As(azaAlias),
			goqu.On(aliasTable.Col(schema.AttrsValuesTableAttributeIDColName).Eq(
				goqu.T(azaAlias).Col(schema.AttrsZoneAttributesTableAttributeIDColName),
			)),
		).Where(goqu.T(azaAlias).Col(schema.AttrsZoneAttributesTableZoneIDColName).Eq(s.ZoneID))
	}

	return sqSelect
}
