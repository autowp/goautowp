package schema

import (
	"time"

	"github.com/doug-martin/goqu/v9"
)

const (
	AttrsUserValuesTableName               = "attrs_user_values"
	AttrsUserValuesTableUserIDColName      = "user_id"
	AttrsUserValuesTableUpdateDateColName  = "update_date"
	AttrsUserValuesTableItemIDColName      = "item_id"
	AttrsUserValuesTableAttributeIDColName = "attribute_id"
	AttrsUserValuesTableAddDateTimeColName = "add_date"
	AttrsUserValuesTableConflictColName    = "conflict"
	AttrsUserValuesTableWeightColName      = "weight"
)

var (
	AttrsUserValuesTable               = goqu.T(AttrsUserValuesTableName)
	AttrsUserValuesTableUserIDCol      = AttrsUserValuesTable.Col(AttrsUserValuesTableUserIDColName)
	AttrsUserValuesTableItemIDCol      = AttrsUserValuesTable.Col(AttrsUserValuesTableItemIDColName)
	AttrsUserValuesTableWeightCol      = AttrsUserValuesTable.Col(AttrsUserValuesTableWeightColName)
	AttrsUserValuesTableAttributeIDCol = AttrsUserValuesTable.Col(AttrsUserValuesTableAttributeIDColName)
	AttrsUserValuesTableConflictCol    = AttrsUserValuesTable.Col(AttrsUserValuesTableConflictColName)
)

type AttrsUserValueRow struct {
	AttributeID int64     `db:"attribute_id"`
	ItemID      int64     `db:"item_id"`
	UserID      int64     `db:"user_id"`
	Conflict    bool      `db:"conflict"`
	UpdateDate  time.Time `db:"update_date"`
	AddDate     time.Time `db:"add_date"`
}
