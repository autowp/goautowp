package schema

import (
	"database/sql"

	"github.com/doug-martin/goqu/v9"
)

const (
	AttrsUserValuesStringTableName               = "attrs_user_values_string"
	AttrsUserValuesStringTableUserIDColName      = "user_id"
	AttrsUserValuesStringTableAttributeIDColName = "attribute_id"
	AttrsUserValuesStringTableItemIDColName      = "item_id"
	AttrsUserValuesStringTableValueColName       = "value"
)

var (
	AttrsUserValuesStringTable               = goqu.T(AttrsUserValuesStringTableName)
	AttrsUserValuesStringTableUserIDCol      = AttrsUserValuesStringTable.Col(AttrsUserValuesStringTableUserIDColName)
	AttrsUserValuesStringTableAttributeIDCol = AttrsUserValuesStringTable.Col(AttrsUserValuesStringTableAttributeIDColName)
	AttrsUserValuesStringTableItemIDCol      = AttrsUserValuesStringTable.Col(AttrsUserValuesStringTableItemIDColName)
	AttrsUserValuesStringTableValueCol       = AttrsUserValuesStringTable.Col(AttrsUserValuesStringTableValueColName)
)

type AttrsUserValuesStringRow struct {
	AttributeID int64          `db:"attribute_id"`
	ItemID      int64          `db:"item_id"`
	UserID      int64          `db:"user_id"`
	Value       sql.NullString `db:"value"`
}
