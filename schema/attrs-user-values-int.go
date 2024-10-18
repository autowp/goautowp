package schema

import (
	"database/sql"

	"github.com/doug-martin/goqu/v9"
)

const (
	AttrsUserValuesIntTableName               = "attrs_user_values_int"
	AttrsUserValuesIntTableUserIDColName      = "user_id"
	AttrsUserValuesIntTableAttributeIDColName = "attribute_id"
	AttrsUserValuesIntTableItemIDColName      = "item_id"
	AttrsUserValuesIntTableValueColName       = "value"
)

var (
	AttrsUserValuesIntTable               = goqu.T(AttrsUserValuesIntTableName)
	AttrsUserValuesIntTableUserIDCol      = AttrsUserValuesIntTable.Col(AttrsUserValuesIntTableUserIDColName)
	AttrsUserValuesIntTableAttributeIDCol = AttrsUserValuesIntTable.Col(AttrsUserValuesIntTableAttributeIDColName)
	AttrsUserValuesIntTableItemIDCol      = AttrsUserValuesIntTable.Col(AttrsUserValuesIntTableItemIDColName)
	AttrsUserValuesIntTableValueCol       = AttrsUserValuesIntTable.Col(AttrsUserValuesIntTableValueColName)
)

type AttrsUserValuesIntRow struct {
	AttributeID int64         `db:"attribute_id"`
	ItemID      int64         `db:"item_id"`
	UserID      int64         `db:"user_id"`
	Value       sql.NullInt32 `db:"value"`
}
